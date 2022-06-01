# Terraform Provider for Pure Storage Fusion

[![GitHub tag (latest SemVer)](https://img.shields.io/github/v/tag/PureStorage-OpenConnect/terraform-provider-fusion?label=release&style=for-the-badge)](https://github.com/PureStorage-OpenConnect/terraform-provider-fusion/releases/latest) [![License](https://img.shields.io/github/license/PureStorage-OpenConnect/terraform-provider-fusion.svg?style=for-the-badge)](LICENSE)

The Terraform Provider for [Fusion][what-is-fusion] is a plugin for Terraform that allows you to interact with Fusion.  Fusion is a Pure Storage product that allows you to provision, manage, and consume enterprise storage with the simple on-demand provisioning, with effortless scale, and self-management of the cloud.  Read more about fusion [here][what-is-fusion]. This provider can be used to manage consumer oriented resources such as volumes, hosts, placement groups, tenant spaces.


Learn More:

* Read the provider [documentation](docs).
* Get help from [Pure Storage Customer Support][customer-support]

## Requirements

* [Terraform 0.15+][terraform-install]

    For general information about Terraform, visit [terraform.io][terraform-install] and [the project][terraform-github] on GitHub.

* Fusion credentials

In order to access the fusion API, you will need the required credentials and configuration.  Namely you will need the `host`, `private_key_file`, and `issuer_id`.  These need to be specified in the provider block in your terraform configuration.  Alternatively, these values can also be supplied using the environment variables: `FUSION_HOST`, `FUSION_PRIVATE_KEY_FILE` and `FUSION_ISSUER_ID`

## Using the provider

It is highly reccomended that you use the pre-built providers.  You should include a block in your terraform config like this:

    terraform {
      required_providers {
        fusion = {
          source  = "PureStorage-OpenConnect/fusion"
          version = "1.0.0"
        }
      }
    }

Then you should be able to just run `terraform init` and it should automatically install the right provider version.  Please check out examples from the [documentation][provider-documentation]  Note: The version number specified here is not the most up-to-date version, please refer to the [documentation][provider-documentation] for the latest version information.

## Getting support

Please don't hesitate to reach out to [Pure Storage Customer Support][customer-support].  If you are having trouble, please try to save and provide the terraform logs.  You can get those logs by setting the `TF_LOG`/`TF_LOG_PATH` envionment variables, for example:

    export TF_LOG=TRACE
    export TF_LOG_PATH=/tmp/terraform-logs
    terraform apply
    <....>
    gzip /tmp/terraform-log

Then the logs will be located at /tmp/terraform-log.gz

## Developing on the provider

If you want to run the tests, you need to set some environment variables.
  - `FUSION_HOST=http://your-fusion-control-plane:8080` This needs to be set to your controlplane endpoint
  - `FUSION_ISSUER_ID=pure1:apikey:abcdefghigjlkmnop` Set this to your fusion issuer ID
  - `FUSION_PRIVATE_KEY_FILE=/tmp/your-fusion-key.pem` Set this to the path of your fusion private key file

Test can be run like normal go tests, for example:

    FUSION_HOST=http://your-fusion-control-plane:8080 FUSION_ISSUER_ID=pure1:apikey:abcdefghigjlkmnop FUSION_PRIVATE_KEY_FILE=/tmp/your-fusion-key.pem make testacc


[terraform-install]: https://www.terraform.io/downloads.html
[terraform-github]: https://github.com/hashicorp/terraform
[provider-documentation]: https://registry.terraform.io/providers/PureStorage-OpenConnect/terraform-provider-fusion/latest/docs
[customer-support]: https://pure1.purestorage.com/support/cases
[what-is-fusion]: https://www.purestorage.com/enable/pure-fusion.html
