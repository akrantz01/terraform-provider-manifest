package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v2"
)

var _ datasource.DataSource = (*fetchDataSource)(nil)

func NewFetchDataSource() datasource.DataSource {
	return &fetchDataSource{}
}

type fetchDataSource struct{}

func (d *fetchDataSource) Metadata(_ context.Context, _ datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "manifest_fetch"
}

func (d *fetchDataSource) GetSchema(context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Description: "The URL used for the request.",
				Type:        types.StringType,
				Computed:    true,
			},
			"url": {
				Description: "The URL for the manifest. Supported schemes are `http` and `https`.",
				Type:        types.StringType,
				Required:    true,
			},
			"filtered_attributes": {
				Description: "The attributes to remove from the manifest.",
				Type: types.ListType{
					ElemType: types.StringType,
				},
				Optional: true,
			},
			"manifests": {
				Description: "The resulting manifests to be applied. Due to a limitation the Terraform Plugin Framework, these must be parsed with `yamldecode` prior to being passed to `kubernetes_manifest`.",
				// TODO: update to `types.Dynamic` pending hashicorp/terraform-plugin-framework#147
				// https://github.com/hashicorp/terraform-plugin-framework/issues/147
				Type: types.ListType{
					ElemType: types.StringType,
				},
				Computed: true,
			},
		},
	}, nil
}

func (d *fetchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model modelV0
	diags := req.Config.Get(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	url := model.URL.Value
	filteredAttributes := parseFilteredAttributes(ctx, model.FilteredAttributes)

	client := &http.Client{}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error creating request", fmt.Sprintf("Error creating request: %s", err))
		return
	}

	response, err := client.Do(request)
	if err != nil {
		resp.Diagnostics.AddError("Error making request", fmt.Sprintf("Error making request: %s", err))
		return
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		resp.Diagnostics.AddError("Received non-success response code", fmt.Sprintf("Received non-success response code: %d", response.StatusCode))
		return
	}

	// Attempt to decode regardless of the content type
	filterableManifests := []map[any]any{}
	if err := unmarshalAllManifests(response.Body, &filterableManifests); err != nil {
		resp.Diagnostics.AddError("Error parsing response body", fmt.Sprintf("Error parsing response body: %s", err))
		return
	}

	// Filter the invalid fields from any manifests
	for _, manifest := range filterableManifests {
		for _, attribute := range filteredAttributes {
			removeAttribute(manifest, attribute)
		}
	}

	// Convert the manifests back to YAML
	var manifests []string
	for _, manifest := range filterableManifests {
		encoded, _ := yaml.Marshal(manifest)
		manifests = append(manifests, string(encoded))
	}

	manifestsState := types.List{}
	diags = tfsdk.ValueFrom(ctx, manifests, types.List{ElemType: types.StringType}.Type(ctx), &manifestsState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	model.ID = types.String{Value: url}
	model.Manifests = manifestsState

	diags = resp.State.Set(ctx, model)
	resp.Diagnostics.Append(diags...)
}

func parseFilteredAttributes(ctx context.Context, raw types.List) [][]string {
	var attributes [][]string

	for _, rawAttribute := range raw.Elems {
		var attribute string
		tfsdk.ValueAs(ctx, rawAttribute, &attribute)

		attributes = append(attributes, strings.Split(attribute, "."))
	}

	return attributes
}

// Unmarshals all manifests in the response
func unmarshalAllManifests(reader io.Reader, manifests *[]map[any]any) error {
	decoder := yaml.NewDecoder(reader)
	decoder.SetStrict(true)

	for {
		var manifest map[any]any

		if err := decoder.Decode(&manifest); err != nil {
			if err != io.EOF {
				return err
			}
			break
		}

		*manifests = append(*manifests, manifest)
	}

	return nil
}

func removeAttribute(manifest map[any]any, path []string) {
	if len(path) == 1 {
		delete(manifest, path[0])
		return
	}

	if sub, ok := manifest[path[0]].(map[any]any); ok {
		removeAttribute(sub, path[1:])
	}
}

type modelV0 struct {
	ID                 types.String `tfsdk:"id"`
	URL                types.String `tfsdk:"url"`
	FilteredAttributes types.List   `tfsdk:"filtered_attributes"`
	Manifests          types.List   `tfsdk:"manifests"`
}
