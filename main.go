package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/drone/drone-plugin-go/plugin"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"
)

var debug bool

type WebHook struct {
	timestamp int64
	images    []string
	namespace string
	source    string
	target    string
	url       string
	token     string
}

type ReqEnvelope struct {
	Verb  string
	Token string
	Json  []byte
	Url   string
}

type Artifact struct {
	ApiVersion string
	Kind       string
	Data       []byte
	Metadata   struct {
		Name string
	}
	Url string
}

func zeroRcs(token string, artifact Artifact) (bool, error) {
	reg, err := regexp.Compile(`"replicas": \d,`)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
	json := reg.ReplaceAllString(string(artifact.Data), "\"replicas\": 0,")

	req := ReqEnvelope{
		Verb:  "PATCH",
		Token: token,
		Url:   fmt.Sprintf("%s/%s", artifact.Url, artifact.Metadata.Name),
		Json:  []byte(json),
	}
	res, err := doRequest(req)
	if err != nil {
		fmt.Printf("%s", err)
	}
	return res, err
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
		contents, err := ioutil.ReadAll(response.Body)
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
	aUrl := fmt.Sprintf("%s/%s", artifact.Url, artifact.Metadata.Name)

	req := ReqEnvelope{
		Url:   aUrl,
		Token: token,
		Verb:  "GET",
	}
	return doRequest(req)

}

func createArtifact(artifact Artifact, token string) {
	param := ReqEnvelope{
		Url:   artifact.Url,
		Token: token,
		Json:  artifact.Data,
		Verb:  "POST",
	}
	doRequest(param)

}

func readArtifactFromFile(workspace string, artifactFile string, apiserver string, namespace string) (Artifact, error) {
	file, e := ioutil.ReadFile(workspace + "/" + artifactFile)
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

func doRequest(param ReqEnvelope) (bool, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	var req *http.Request
	var err error
	// post payload to each artifact
	if param.Json != nil {
		req, err = http.NewRequest(param.Verb, param.Url, nil)
	} else {
		req, err = http.NewRequest(param.Verb, param.Url, bytes.NewBuffer(param.Json))
	}

	if debug {
		fmt.Println("HTTP Request %s", param.Verb)
		fmt.Println("HTTP Request %s", param.Url)
		fmt.Println("HTTP Request %s", string(param.Json))
	}

	req.Header.Set("Authorization", "Bearer "+param.Token)
	response, err := client.Do(req)
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	} else {
		defer response.Body.Close()
		if debug {
			contents, err := ioutil.ReadAll(response.Body)
			if err != nil {
				os.Exit(1)
			}
			fmt.Printf("%s\n", string(contents))
		}
		if response.StatusCode == 200 {
			return true, err
		}
	}
	return false, err
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func sendWebhook(wh WebHook) {

	jwh, err := json.Marshal(wh)
	if err != nil {
		fmt.Println(err)
		return
	}
	req := ReqEnvelope{
		Verb:  "POST",
		Token: wh.token,
		Url:   wh.url,
		Json:  []byte(jwh),
	}
	doRequest(req)
}

var deployments []string

func main() {
	var vargs = struct {
		ReplicationControllers []string `json:replicationcontrollers`
		Services               []string `json:services`
		ApiServer              string   `json:apiserver`
		Token                  string   `json:token`
		Namespace              string   `json:namespace`
		Debug                  string   `json:debug`
		Webhook                string   `json:webhook`
		Source                 string   `json:source`
		WebHookToken           string   `json:webhook_token`
	}{}

	workspace := plugin.Workspace{}
	plugin.Param("workspace", &workspace)
	plugin.Param("vargs", &vargs)
	plugin.Parse()

	// Iterate over rcs and svcs
	for _, rc := range vargs.ReplicationControllers {
		artifact, e := readArtifactFromFile(&workspace, rc, vargs.ApiServer, vargs.Namespace)
		if b, _ := existsArtifact(artifact, token); b {
			deleteArtifact(artifact, token)
		}
		createArtifact(artifact, vargs.Token)
	}
	for _, rc := range vargs.Services {
		createArtifact(artifact, vargs.Token)
	}
	wh := WebHook{
		timestamp: makeTimestamp(),
		images:    deployments,
		namespace: vargs.Namespace,
		source:    vargs.Source,
		target:    vargs.ApiServer,
		url:       vargs.Webhook,
		token:     vargs.WebHookToken,
	}
	sendWebhook(wh)
}
