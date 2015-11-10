package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/drone/drone-plugin-go/plugin"
	"io/ioutil"
	"net/http"
	"os"
)

type Artifact struct {
	ApiVersion string
	Kind       string
	Data       []byte
	Metadata   struct {
		Name string
	}
	Url string
}

// Kubernetes API doesn't delete pods when deleting an RC
// to cleanly remove the rc we have to set `replicas=0`
// and then delete the RC
func deleteArtifact(artifact Artifact, token string) (bool, error) {
	url := fmt.Sprintf("%s/%s", artifact.Url, artifact.Metadata.Name)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	// post payload to each artifact
	req, err := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := client.Do(req)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			os.Exit(1)
		}
		fmt.Printf("%s\n", string(contents))
		if response.StatusCode == 200 {
			return true, err
		}
	}
	return false, err
}

func existsArtifact(artifact Artifact, token string) (bool, error) {
	url := fmt.Sprintf("%s/%s", artifact.Url, artifact.Metadata.Name)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	// post payload to each artifact
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response, err := client.Do(req)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	} else {
		defer response.Body.Close()
		if response.StatusCode == 200 {
			return true, err
		}
	}
	return false, err
}

func createArtifact(artifact Artifact, token string) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// post payload to each artifact
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(artifact.Data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		os.Exit(1)
	}
	fmt.Printf("%s\n", string(contents))

}

func readArtifactFromFile(workspace string, artifact string, apiserver string, namespace string) (Artifact, error) {
	file, e := ioutil.ReadFile(workspace + "/" + artifact)
	// fmt.Println(string(file))
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
	}
	artifact := Artifact{}
	json.Unmarshal(file, &artifact)
	artifact.Data = file
	if artifact.Kind == "ReplicationController" {
		artifact.Url = fmt.Sprintf("%s/api/v1/namespaces/%s/replicationcontrollers", apiserver, namespace)
	}
	if artifact.Kind == "Service" {
		artifact.Url = fmt.Sprintf("%s/api/v1/namespaces/%s/services", apiserver, namespace)
	}

	return artifact, e
}

func main() {
	var vargs = struct {
		ReplicationControllers []string `json:replicationcontrollers`
		Services               []string `json:services`
		ApiServer              string   `json:apiserver`
		Token                  string   `json:token`
		Namespace              string   `json:namespace`
	}{}

	workspace := plugin.Workspace{}
	plugin.Param("workspace", &workspace)
	plugin.Param("vargs", &vargs)
	plugin.Parse()

	// Iterate over rcs and svcs
	for _, rc := range vargs.ReplicationControllers {
		artifact, e := readArtifactFromFile(workspace, rc, vargs.ApiServer, vargs.Namespace)
		if b, _ := existsArtifact(artifact, token); b {
			deleteArtifact(artifact, token)
		}
		createArtifact(artifact, vargs.Token)
	}
	for _, rc := range vargs.Services {
		createArtifact(artifact, vargs.Token)
	}
}
