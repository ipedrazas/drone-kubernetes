package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin"
	"io/ioutil"
	"net/http"
	"os"
)

func createArtifact(artifact string, url string, token string) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	file, e := ioutil.ReadFile(artifact)
	if e != nil {
		os.Exit(1)
	}
	// post payload to each artifact
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(file))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	// contents, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	// 	os.Exit(1)
	// }
	// fmt.Printf("%s\n", string(contents))

}

func main() {
	var repo = drone.Repo{}
	var build = drone.Build{}
	var vargs = struct {
		ReplicationControllers []string `json:"replicationcontrollers"`
		Services               []string `json:"services"`
		ApiServer              string   `json:apiserver`
		Token                  string   `json:token`
		Namespace              string   `json:namespace`
	}{}

	plugin.Param("repo", &repo)
	plugin.Param("build", &build)
	plugin.Param("vargs", &vargs)
	plugin.Parse()

	rc_url := fmt.Sprintf("%s/api/v1/namespaces/%s/replicationcontrollers", vargs.ApiServer, vargs.Namespace)
	svc_url := fmt.Sprintf("%s/api/v1/namespaces/%s/services", vargs.ApiServer, vargs.Namespace)

	// Iterate over rcs and svcs
	for _, rc := range vargs.ReplicationControllers {
		createArtifact(rc, rc_url, vargs.Token)
	}
	for _, rc := range vargs.Services {
		createArtifact(rc, svc_url, vargs.Token)
	}
}