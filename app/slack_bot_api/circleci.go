package main

import (
	"bitbucket.org/wanocoltd/vk_infra/app/modules/common"
	"bitbucket.org/wanocoltd/vkgo_core/v2/util/log"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type CircleCIBuildRequest struct {
	Branch     string      `json:"branch"`
	Parameters *Parameters `json:"parameters"`
}

type Parameters struct {
	IsStageDeploy bool `json:"is_stage_deploy"`
	IsProdDeploy  bool `json:"is_prod_deploy"`
}

type StartDeployRequest struct {
	owner, repoName, branch, env, circleciToken string
}

const BASE_CIRCLECI_BUILD_URL = "https://circleci.com/api/v2/project/bitbucket/%v/%v/pipeline"

func startDeploy(depReq *StartDeployRequest) error {
	circleCIBuildReq := &CircleCIBuildRequest{
		Branch:     depReq.branch,
		Parameters: &Parameters{},
	}
	switch common.VKENV(depReq.env) {
	case common.ENV_VK_STAGE:
		circleCIBuildReq.Parameters.IsStageDeploy = true
	case common.ENV_VK_PROD:
		circleCIBuildReq.Parameters.IsProdDeploy = true
	default:
		return errors.New("invalid env")
	}
	reqJson, err := json.Marshal(circleCIBuildReq)
	if err != nil {
		log.Error(err)
		return err
	}
	endpoint := fmt.Sprintf(BASE_CIRCLECI_BUILD_URL, depReq.owner, depReq.repoName)
	req, err := http.NewRequest(
		"POST",
		endpoint,
		bytes.NewReader(reqJson),
	)
	if err != nil {
		return err
	}

	// Content-Type 設定
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(depReq.circleciToken, "")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return err
}
