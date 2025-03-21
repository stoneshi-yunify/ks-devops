/*
Copyright 2020 KubeSphere Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package jenkins

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"k8s.io/klog/v2"

	"github.com/kubesphere/ks-devops/pkg/client/devops"
)

type Pipeline struct {
	HttpParameters *devops.HttpParameters
	Jenkins        *Jenkins
	Path           string
}

const (
	GetPipelineUrl         = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/"
	ListPipelinesUrl       = "/blue/rest/search/?"
	GetPipelineRunUrl      = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/runs/%s/"
	ListPipelineRunUrl     = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/runs/?"
	StopPipelineUrl        = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/runs/%s/stop/?"
	ReplayPipelineUrl      = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/runs/%s/replay/"
	RunPipelineUrl         = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/runs/"
	GetArtifactsUrl        = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/runs/%s/artifacts/?"
	GetRunLogUrl           = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/runs/%s/log/?"
	GetStepLogUrl          = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/runs/%s/nodes/%s/steps/%s/log/?"
	GetPipelineRunNodesUrl = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/runs/%s/nodes/?"
	SubmitInputStepUrl     = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/runs/%s/nodes/%s/steps/%s/"
	GetNodeStepsUrl        = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/runs/%s/nodes/%s/steps/?"

	GetBranchPipelineUrl     = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/branches/%s/"
	GetBranchPipelineRunUrl  = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/branches/%s/runs/%s/"
	StopBranchPipelineUrl    = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/branches/%s/runs/%s/stop/?"
	ReplayBranchPipelineUrl  = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/branches/%s/runs/%s/replay/"
	RunBranchPipelineUrl     = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/branches/%s/runs/"
	GetBranchArtifactsUrl    = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/branches/%s/runs/%s/artifacts/?"
	GetBranchRunLogUrl       = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/branches/%s/runs/%s/log/?"
	GetBranchStepLogUrl      = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/branches/%s/runs/%s/nodes/%s/steps/%s/log/?"
	GetBranchNodeStepsUrl    = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/branches/%s/runs/%s/nodes/%s/steps/?"
	GetBranchPipeRunNodesUrl = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/branches/%s/runs/%s/nodes/?"
	CheckBranchPipelineUrl   = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/branches/%s/runs/%s/nodes/%s/steps/%s/"
	GetPipeBranchUrl         = "/blue/rest/organizations/jenkins/pipelines/%s/pipelines/%s/branches/?"
	ScanBranchUrl            = "/job/%s/job/%s/build?"
	GetConsoleLogUrl         = "/job/%s/job/%s/indexing/consoleText"
	GetCrumbUrl              = "/crumbIssuer/api/json/"
	GetSCMServersUrl         = "/blue/rest/organizations/jenkins/scm/%s/servers/"
	GetSCMOrgUrl             = "/blue/rest/organizations/jenkins/scm/%s/organizations/?"
	GetOrgRepoUrl            = "/blue/rest/organizations/jenkins/scm/%s/organizations/%s/repositories/?"
	CreateSCMServersUrl      = "/blue/rest/organizations/jenkins/scm/%s/servers/"
	ValidateUrl              = "/blue/rest/organizations/jenkins/scm/%s/validate"

	GetNotifyCommitUrl = "/git/notifyCommit/?"
	GithubWebhookUrl   = "/github-webhook/"
	// GenericWebhookUrl comes from Jenkins plugin, see also https://github.com/jenkinsci/generic-webhook-trigger-plugin
	GenericWebhookUrl     = "/generic-webhook-trigger/invoke?"
	CheckScriptCompileUrl = "/job/%s/job/%s/descriptorByName/org.jenkinsci.plugins.workflow.cps.CpsFlowDefinition/checkScriptCompile"

	CheckPipelienCronUrl = "/job/%s/job/%s/descriptorByName/hudson.triggers.TimerTrigger/checkSpec?%s"
	CheckCronUrl         = "/job/%s/descriptorByName/hudson.triggers.TimerTrigger/checkSpec?%s"

	cronJobLayout = "Monday, January 2, 2006 15:04:05 PM"

	CheckPipelineName = "/job/%s/checkJobName?"
)

func (p *Pipeline) CheckPipelineName() (map[string]interface{}, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		return nil, err
	}

	/*
		if the name exist , the res will be
		 <div class=error><img src='/static/7abb227e/images/none.gif' height=16 width=1>A job already exists with the name ‘devopsd6b97’</div>
	*/
	pattern := `>[^<]+<`

	reg := regexp.MustCompile(pattern)

	result := make(map[string]interface{})

	match := reg.FindString(string(res))
	if match != "" {
		result["exist"] = true
	} else {
		result["exist"] = false
	}

	return result, nil
}

func (p *Pipeline) GetPipeline() (*devops.Pipeline, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	var pipeline devops.Pipeline

	err = json.Unmarshal(res, &pipeline)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal JSON data to Pipeline, raw: %s, error: %v", res, err)
		return nil, err
	}
	return &pipeline, err
}

func (p *Pipeline) ListPipelines() (*devops.PipelineList, error) {
	res, _, err := p.Jenkins.SendPureRequestWithHeaderResp(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		if jErr, ok := err.(*JkError); ok {
			switch jErr.Code {
			case 404:
				err = fmt.Errorf("please check if there're any Jenkins plugins issues exist")
			default:
				err = fmt.Errorf("please check if Jenkins is running well")
			}
			klog.Errorf("API '%s' request response code is '%d'", p.Path, jErr.Code)
		} else {
			err = fmt.Errorf("unknow errors happend when communicate with Jenkins")
		}
		return nil, err
	}
	count, err := p.searchPipelineCount()
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	pipelienList, err := devops.UnmarshalPipeline(count, res)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	return pipelienList, err
}

func (p *Pipeline) searchPipelineCount() (int, error) {
	query, _ := ParseJenkinsQuery(p.HttpParameters.Url.RawQuery)
	query.Set("start", "0")
	query.Set("limit", "1000")
	query.Set("depth", "-1")

	formatUrl := ListPipelinesUrl + query.Encode()

	res, err := p.Jenkins.SendPureRequest(formatUrl, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return 0, err
	}
	var pipelines []devops.Pipeline
	err = json.Unmarshal(res, &pipelines)
	if err != nil {
		klog.Error(err)
		return 0, err
	}
	return len(pipelines), nil
}

func (p *Pipeline) GetPipelineRun() (*devops.PipelineRun, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	var pipelineRun devops.PipelineRun
	err = json.Unmarshal(res, &pipelineRun)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	return &pipelineRun, err
}

func (p *Pipeline) ListPipelineRuns() (*devops.PipelineRunList, error) {
	// prefer to use listPipelineRunsByRemotePaging once the corresponding issues from BlueOcean fixed
	return p.listPipelineRunsByLocalPaging()
}

// listPipelineRunsByRemotePaging get the pipeline runs with pagination by remote (Jenkins BlueOcean plugin)
// get the pagination information from the server side is better than the local side, but the API has some issues
// see also https://github.com/kubesphere/kubesphere/issues/3507
func (p *Pipeline) listPipelineRunsByRemotePaging() (*devops.PipelineRunList, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	var pipelineRunList devops.PipelineRunList
	err = json.Unmarshal(res, &pipelineRunList.Items)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	total, err := p.searchPipelineRunsCount()
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	pipelineRunList.Total = total
	return &pipelineRunList, err
}

// listPipelineRunsByLocalPaging should be a temporary solution
// see also https://github.com/kubesphere/kubesphere/issues/3507
func (p *Pipeline) listPipelineRunsByLocalPaging() (runList *devops.PipelineRunList, err error) {
	desiredStart, desiredLimit := p.parsePaging()

	var pageUrl *url.URL // get all Pipeline runs
	if pageUrl, err = p.resetPaging(0, 10000); err != nil {
		return
	}
	res, err := p.Jenkins.SendPureRequest(pageUrl.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	runList = &devops.PipelineRunList{
		Items: make([]devops.PipelineRun, 0),
	}
	if err = json.Unmarshal(res, &runList.Items); err != nil {
		klog.Error(err)
		return nil, err
	}

	// set the total count number
	runList.Total = len(runList.Items)

	// keep the desired data/
	if desiredStart+1 >= runList.Total {
		// beyond the boundary, return an empty
		return
	}

	endIndex := runList.Total
	if desiredStart+desiredLimit < endIndex {
		endIndex = desiredStart + desiredLimit
	}
	runList.Items = runList.Items[desiredStart:endIndex]
	return
}

// resetPaging reset the paging setting from request, support start, limit
func (p *Pipeline) resetPaging(start, limit int) (path *url.URL, err error) {
	query, _ := ParseJenkinsQuery(p.HttpParameters.Url.RawQuery)
	query.Set("start", strconv.Itoa(start))
	query.Set("limit", strconv.Itoa(limit))
	p.HttpParameters.Url.RawQuery = query.Encode()
	path, err = url.Parse(p.Path)
	return
}

func (p *Pipeline) parsePaging() (start, limit int) {
	query, _ := ParseJenkinsQuery(p.HttpParameters.Url.RawQuery)
	start, _ = strconv.Atoi(query.Get("start"))
	limit, _ = strconv.Atoi(query.Get("limit"))
	return
}

func (p *Pipeline) searchPipelineRunsCount() (int, error) {
	u, err := p.resetPaging(0, 1000)
	if err != nil {
		return 0, err
	}
	res, err := p.Jenkins.SendPureRequest(u.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return 0, err
	}
	var runs []devops.PipelineRun
	err = json.Unmarshal(res, &runs)
	if err != nil {
		klog.Error(err)
		return 0, err
	}
	return len(runs), nil
}

func (p *Pipeline) StopPipeline() (*devops.StopPipeline, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	var stopPipeline devops.StopPipeline
	err = json.Unmarshal(res, &stopPipeline)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &stopPipeline, err
}

func (p *Pipeline) ReplayPipeline() (*devops.ReplayPipeline, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	var replayPipeline devops.ReplayPipeline
	err = json.Unmarshal(res, &replayPipeline)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &replayPipeline, err
}

func (p *Pipeline) RunPipeline() (*devops.RunPipeline, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	var runPipeline devops.RunPipeline
	err = json.Unmarshal(res, &runPipeline)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &runPipeline, err
}

func (p *Pipeline) GetArtifacts() ([]devops.Artifacts, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	var artifacts []devops.Artifacts
	err = json.Unmarshal(res, &artifacts)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return artifacts, err
}

func (p *Pipeline) GetRunLog() ([]byte, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
	}

	return res, err
}

func (p *Pipeline) GetStepLog() ([]byte, http.Header, error) {
	res, header, err := p.Jenkins.SendPureRequestWithHeaderResp(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
	}

	return res, header, err
}

func (p *Pipeline) GetNodeSteps() ([]devops.NodeSteps, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
	}

	var nodeSteps []devops.NodeSteps
	err = json.Unmarshal(res, &nodeSteps)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return nodeSteps, err
}

func (p *Pipeline) GetPipelineRunNodes() ([]devops.PipelineRunNodes, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	var pipelineRunNodes []devops.PipelineRunNodes
	err = json.Unmarshal(res, &pipelineRunNodes)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return pipelineRunNodes, err
}

func (p *Pipeline) SubmitInputStep() ([]byte, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
	}

	return res, err
}

func (p *Pipeline) GetBranchPipeline() (*devops.BranchPipeline, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	var branchPipeline devops.BranchPipeline
	err = json.Unmarshal(res, &branchPipeline)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &branchPipeline, err
}

func (p *Pipeline) GetBranchPipelineRun() (*devops.PipelineRun, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	var branchPipelineRun devops.PipelineRun
	err = json.Unmarshal(res, &branchPipelineRun)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &branchPipelineRun, err
}

func (p *Pipeline) StopBranchPipeline() (*devops.StopPipeline, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	var branchStopPipeline devops.StopPipeline
	err = json.Unmarshal(res, &branchStopPipeline)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &branchStopPipeline, err
}

func (p *Pipeline) ReplayBranchPipeline() (*devops.ReplayPipeline, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	var branchReplayPipeline devops.ReplayPipeline
	err = json.Unmarshal(res, &branchReplayPipeline)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &branchReplayPipeline, err
}

func (p *Pipeline) RunBranchPipeline() (*devops.RunPipeline, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	var branchRunPipeline devops.RunPipeline
	err = json.Unmarshal(res, &branchRunPipeline)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &branchRunPipeline, err
}

func (p *Pipeline) GetBranchArtifacts() ([]devops.Artifacts, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	var artifacts []devops.Artifacts
	err = json.Unmarshal(res, &artifacts)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return artifacts, err
}

func (p *Pipeline) GetBranchRunLog() ([]byte, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
	}

	return res, err
}

func (p *Pipeline) GetBranchStepLog() ([]byte, http.Header, error) {
	res, header, err := p.Jenkins.SendPureRequestWithHeaderResp(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
	}

	return res, header, err
}

func (p *Pipeline) GetBranchNodeSteps() ([]devops.NodeSteps, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	var branchNodeSteps []devops.NodeSteps
	err = json.Unmarshal(res, &branchNodeSteps)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return branchNodeSteps, err
}

func (p *Pipeline) GetBranchPipelineRunNodes() ([]devops.BranchPipelineRunNodes, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	var branchPipelineRunNodes []devops.BranchPipelineRunNodes
	err = json.Unmarshal(res, &branchPipelineRunNodes)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return branchPipelineRunNodes, err
}

func (p *Pipeline) SubmitBranchInputStep() ([]byte, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
	}

	return res, err
}

func (p *Pipeline) GetPipelineBranch() (*devops.PipelineBranch, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	var pipelineBranch devops.PipelineBranch
	err = json.Unmarshal(res, &pipelineBranch)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &pipelineBranch, err
}

func (p *Pipeline) ScanBranch() ([]byte, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
	}

	return res, err
}

func (p *Pipeline) GetConsoleLog() ([]byte, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
	}

	return res, err
}

func (p *Pipeline) GetCrumb() (*devops.Crumb, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	var crumb devops.Crumb
	err = json.Unmarshal(res, &crumb)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &crumb, err
}

func (p *Pipeline) GetSCMServers() ([]devops.SCMServer, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	var SCMServer []devops.SCMServer
	err = json.Unmarshal(res, &SCMServer)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return SCMServer, err
}

func (p *Pipeline) GetSCMOrg() ([]devops.SCMOrg, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	var SCMOrg []devops.SCMOrg
	err = json.Unmarshal(res, &SCMOrg)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return SCMOrg, err
}

func (p *Pipeline) GetOrgRepo() (devops.OrgRepo, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return devops.OrgRepo{}, err
	}
	var OrgRepo devops.OrgRepo
	err = json.Unmarshal(res, &OrgRepo)
	if err != nil {
		klog.Error(err)
		return devops.OrgRepo{}, err
	}

	return OrgRepo, err
}

func (p *Pipeline) CreateSCMServers() (*devops.SCMServer, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}
	var SCMServer devops.SCMServer
	err = json.Unmarshal(res, &SCMServer)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &SCMServer, err
}

func (p *Pipeline) GetNotifyCommit() ([]byte, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
	}

	return res, err
}

func (p *Pipeline) GithubWebhook() ([]byte, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
	}

	return res, err
}

func (p *Pipeline) GenericWebhook() ([]byte, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
	}

	return res, err
}

func (p *Pipeline) Validate() (*devops.Validates, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	var validates devops.Validates
	err = json.Unmarshal(res, &validates)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &validates, err
}

func (p *Pipeline) CheckScriptCompile() (*devops.CheckScript, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	// Jenkins will return different struct according to different results.
	var checkScript devops.CheckScript
	ok := json.Unmarshal(res, &checkScript)
	if ok != nil {
		var resJson []*devops.CheckScript
		err := json.Unmarshal(res, &resJson)
		if err != nil {
			klog.Error(err)
			return nil, err
		}

		return resJson[0], nil
	}

	return &checkScript, err

}

func (p *Pipeline) CheckCron() (*devops.CheckCronRes, error) {
	var res = new(devops.CheckCronRes)

	reqJenkins := &http.Request{
		Method: http.MethodGet,
		Header: p.HttpParameters.Header,
	}
	if cronServiceURL, err := url.Parse(p.Jenkins.Server + p.Path); err != nil {
		klog.Errorf(fmt.Sprintf("cannot parse Jenkins cronService URL, error: %#v", err))
		return interanlErrorMessage(), err
	} else {
		reqJenkins.URL = cronServiceURL
	}

	client := &http.Client{Timeout: 30 * time.Second}
	reqJenkins.SetBasicAuth(p.Jenkins.Requester.BasicAuth.Username, p.Jenkins.Requester.BasicAuth.Password)
	resp, err := client.Do(reqJenkins)
	if err != nil {
		klog.Error(err)
		return interanlErrorMessage(), err
	}

	var responseText string
	if resp != nil {
		if responseData, err := getRespBody(resp); err == nil {
			responseText = string(responseData)
		} else {
			klog.Error(err)
			return interanlErrorMessage(), fmt.Errorf("cannot get the response body from the Jenkins cron service request, %#v", err)
		}

		defer func() {
			_ = resp.Body.Close()
		}()

		statusCode := resp.StatusCode
		if statusCode != http.StatusOK && statusCode != http.StatusBadRequest {
			// the response body is meaningless for the users, but it's useful for a contributor
			klog.Errorf("cron service from Jenkins is unavailable, error response: %v, status code: %d", responseText, statusCode)
			return interanlErrorMessage(), err
		}
	}
	klog.V(8).Infof("response text: %s", responseText)

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader([]byte(responseText)))
	if err != nil {
		klog.Error(err)
		return interanlErrorMessage(), err
	}
	// it gives a message which located in <div>...</div> when the status code is 200
	doc.Find("div").Each(func(i int, selection *goquery.Selection) {
		res.Message = selection.Text()
		res.Result, _ = selection.Attr("class")
	})
	// it gives a message which located in <pre>...</pre> when the status code is 400
	doc.Find("pre").Each(func(i int, selection *goquery.Selection) {
		res.Message = selection.Text()
		res.Result = "error"
	})
	if res.Result == "ok" {
		res.LastTime, res.NextTime, err = parseCronJobTime(res.Message)
		if err != nil {
			klog.Error(err)
			return interanlErrorMessage(), err
		}
	}

	return res, err
}

func interanlErrorMessage() *devops.CheckCronRes {
	return &devops.CheckCronRes{
		Result:  "error",
		Message: "internal errors happened, get more details by checking ks-apiserver log output",
	}
}

func parseCronJobTime(msg string) (string, string, error) {
	msg = strings.ReplaceAll(msg, "Coordinated Universal Time", "UTC")
	msg = strings.ReplaceAll(msg, " at ", " ")
	times := strings.Split(msg, ";")

	lastTmp := strings.Split(times[0], " ")
	lastCount := len(lastTmp)
	lastTmp = lastTmp[lastCount-7 : lastCount-1]
	lastTime := strings.Join(lastTmp, " ")
	lastUinx, err := time.Parse(cronJobLayout, lastTime)
	if err != nil {
		klog.Error(err)
		return "", "", err
	}
	last := lastUinx.Format(time.RFC3339)

	nextTmp := strings.Split(times[1], " ")
	nextCount := len(nextTmp)
	nextTmp = nextTmp[nextCount-7 : nextCount-1]
	nextTime := strings.Join(nextTmp, " ")
	nextUinx, err := time.Parse(cronJobLayout, nextTime)
	if err != nil {
		klog.Error(err)
		return "", "", err
	}
	next := nextUinx.Format(time.RFC3339)

	return last, next, nil
}

func (p *Pipeline) ToJenkinsfile() (*devops.ResJenkinsfile, error) {
	res, err := p.Jenkins.SendPureRequest(p.Path, p.HttpParameters)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	var jenkinsfile devops.ResJenkinsfile
	err = json.Unmarshal(res, &jenkinsfile)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	return &jenkinsfile, err
}

func (p *Pipeline) ToJSON() (result map[string]interface{}, err error) {
	var data []byte
	if data, err = p.Jenkins.SendPureRequest(p.Path, p.HttpParameters); err == nil {
		err = json.Unmarshal(data, &result)
	}
	return
}
