package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

var _ provider.Provider = (*manifestProvider)(nil)
var _ provider.ProviderWithMetadata = (*manifestProvider)(nil)

type manifestProvider struct {
	version string
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &manifestProvider{
			version: version,
		}
	}
}

func (p *manifestProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "manifest"
	resp.Version = p.version
}

func (p *manifestProvider) GetSchema(context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{}, nil
}

func (p *manifestProvider) Configure(context.Context, provider.ConfigureRequest, *provider.ConfigureResponse) {
}

func (p *manifestProvider) DataSources(context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewFetchDataSource,
	}
}

func (p *manifestProvider) Resources(context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}
