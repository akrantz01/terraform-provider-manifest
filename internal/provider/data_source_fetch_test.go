package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestDataSource_SingleDocument_Unfiltered(t *testing.T) {
	server := setupMockServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(unfilteredResourceStatement, server.URL, "single"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.manifest_fetch.test", "manifests.#", "1"),
					resource.TestCheckResourceAttr("data.manifest_fetch.test", "manifests.0", singleDocument),
				),
			},
		},
	})
}

func TestDataSource_MultipleDocuments_Unfiltered(t *testing.T) {
	server := setupMockServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(unfilteredResourceStatement, server.URL, "multiple"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.manifest_fetch.test", "manifests.#", "3"),
					resource.TestCheckResourceAttr("data.manifest_fetch.test", "manifests.0", multipleDocument1),
					resource.TestCheckResourceAttr("data.manifest_fetch.test", "manifests.1", multipleDocument2),
					resource.TestCheckResourceAttr("data.manifest_fetch.test", "manifests.2", multipleDocument3),
				),
			},
		},
	})
}

func TestDataSource_SingleDocument_Filtered(t *testing.T) {
	server := setupMockServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(filteredResourceStatement, server.URL, "single"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.manifest_fetch.test", "manifests.#", "1"),
					resource.TestCheckResourceAttr("data.manifest_fetch.test", "manifests.0", "apiVersion: testing.k8s.io/v1\nkind: Test\nmetadata:\n  annotations:\n    hello: world\nspec:\n  some: key\n"),
				),
			},
		},
	})
}

func TestDataSource_MultipleDocuments_Filtered(t *testing.T) {
	server := setupMockServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(filteredResourceStatement, server.URL, "multiple"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.manifest_fetch.test", "manifests.#", "3"),
					resource.TestCheckResourceAttr("data.manifest_fetch.test", "manifests.0", "apiVersion: testing.k8s.io/v1\nkind: Test\n"),
					resource.TestCheckResourceAttr("data.manifest_fetch.test", "manifests.1", "apiVersion: testing.k8s.io/v1\nkind: test\nmetadata: {}\n"),
					resource.TestCheckResourceAttr("data.manifest_fetch.test", "manifests.2", multipleDocument3)),
			},
		},
	})
}

func TestDataSource_OnlyResources(t *testing.T) {
	server := setupMockServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(onlyResourcesStatement, server.URL, "multiple"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.manifest_fetch.test", "manifests.#", "1"),
					resource.TestCheckResourceAttr("data.manifest_fetch.test", "manifests.0", multipleDocument1)),
			},
		},
	})
}

func TestDataSource_Failure(t *testing.T) {
	server := setupMockServer()
	defer server.Close()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories(),
		Steps: []resource.TestStep{
			{
				Config:      fmt.Sprintf(unfilteredResourceStatement, server.URL, "failure"),
				ExpectError: regexp.MustCompile("Received non-success response code: 500"),
			},
		},
	})
}

func setupMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/single":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(singleDocument))
		case "/multiple":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(multipleDocuments))
		case "/failure":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("error"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

const singleDocument = `apiVersion: testing.k8s.io/v1
kind: Test
metadata:
  annotations:
    hello: world
  creationTimestamp: null
spec:
  some: key
status:
  abc: def
  bool: true
`

const multipleDocument1 = `apiVersion: testing.k8s.io/v1
kind: Test
status: hello
`
const multipleDocument2 = `apiVersion: testing.k8s.io/v1
kind: test
metadata:
  creationTimestamp: null
`
const multipleDocument3 = `apiVersion: testing.k8s.io/v1
kind: test
spec:
  un: changed
`

var multipleDocuments = strings.Join([]string{multipleDocument1, multipleDocument2, multipleDocument3}, "\n---\n")

const unfilteredResourceStatement = `
data "manifest_fetch" "test" {
	url = "%s/%s"
}`

const filteredResourceStatement = `
data "manifest_fetch" "test" {
	url = "%s/%s"
	filtered_attributes = [
		"status",
		"metadata.creationTimestamp",
	]
}`

const onlyResourcesStatement = `
data "manifest_fetch" "test" {
	url = "%s/%s"
	only_resources = [
		"testing.k8s.io/v1/Test"
	]
}
`
