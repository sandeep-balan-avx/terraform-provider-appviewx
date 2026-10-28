# appviewx_push_gcp_certificate_manager

Pushes a certificate managed by AppViewX to **GCP Certificate Manager** and, optionally, binds it to a GCP Load Balancer via a Certificate Map.

The resource calls `POST /avxapi/certificate/pushToDevice`, which is intercepted by `ApplicationConnectorSaveAndPushTransformer` and routed to the `save-application-connector` action. AppViewX then triggers a push workflow via `CertGCPPushHelper`.

Whether a push-only or push-and-bind operation occurs is determined by the **AppViewX DeviceProfile** selected in `selected_profiles`. Use a "Certificate Manager" profile for push-only, or a "Load Balancer" profile together with the `certificate_map_*` fields for LB binding.

## Example – Push only

```hcl
resource "appviewx_push_gcp_certificate_manager" "example" {
  certificate_id     = appviewx_create_certificate.this.resource_id
  certificate_name   = "dev-api-example-com"
  project            = "my-gcp-project"
  location           = "us-central1"
  gcp_connector_name = "GCP connector"
  selected_profiles  = ["GCP.Single.All:my-gcp-project:Certificate Manager"]

  wait_for_completion   = true
  wait_timeout_seconds  = 600
  poll_interval_seconds = 10
}
```

## Example – Push and bind to a Load Balancer

```hcl
resource "appviewx_push_gcp_certificate_manager" "example_lb" {
  certificate_id     = appviewx_create_certificate.this.resource_id
  certificate_name   = "dev-api-example-com"
  project            = "my-gcp-project"
  location           = "us-central1"
  gcp_connector_name = "GCP connector"
  selected_profiles  = ["GCP.Single.All:my-gcp-project:Load Balancer"]

  # Certificate Map / Load Balancer binding
  certificate_map_name             = "dev-api-cert-map"
  is_new_certificate_map           = true
  certificate_map_entry_name       = "dev-api-cert-map-entry"
  is_new_certificate_map_entry     = true
  is_primary_certificate_map_entry = false
  hostname                         = "dev.api.example.com"

  wait_for_completion   = true
  wait_timeout_seconds  = 600
  poll_interval_seconds = 10
}
```

## Argument Reference

### Required

| Name | Type | Description |
|------|------|-------------|
| `certificate_id` | string | AppViewX certificate resource id (`resource_id` from `appviewx_create_certificate`). |
| `certificate_name` | string | Name of the certificate in GCP Certificate Manager. |
| `project` | string | GCP project id. |
| `location` | string | GCP location: a region (e.g. `us-central1`) or `global`. |
| `gcp_connector_name` | string | AppViewX GCP connector name. |
| `selected_profiles` | list(string) | AppViewX DeviceProfile IDs. Use a `Certificate Manager` profile for push-only, or a `Load Balancer` profile for LB binding. |

### Optional – Basic

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `is_new_certificate` | bool | `true` | `true` to create the cert in GCP CM; `false` to update an existing one. |
| `push_automatically` | bool | `true` | Trigger the push immediately. |

### Optional – Certificate Map / Load Balancer binding

Set `certificate_map_name` to enable LB binding. All other fields in this group are only used when `certificate_map_name` is set.

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `certificate_map_name` | string | `""` | GCP Certificate Map name. Required for LB binding. |
| `is_new_certificate_map` | bool | `true` | `true` to create a new Certificate Map. |
| `certificate_map_entry_name` | string | `""` | GCP Certificate Map Entry name. |
| `is_new_certificate_map_entry` | bool | `true` | `true` to create a new entry; `false` to update. |
| `is_primary_certificate_map_entry` | bool | `true` | `true` for a PRIMARY matcher; `false` for a hostname matcher. |
| `hostname` | string | `""` | Hostname for the entry when `is_primary_certificate_map_entry` is `false`. |

### Optional – Wait for completion

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `wait_for_completion` | bool | `true` | Block until the GCP push workflow completes. |
| `wait_timeout_seconds` | int | `600` | Maximum seconds to wait. |
| `poll_interval_seconds` | int | `10` | Seconds between poll attempts. |

## Attribute Reference

| Name | Type | Description |
|------|------|-------------|
| `certificate_manager_certificate_id` | string | GCP Certificate Manager resource id: `projects/{project}/locations/{location}/certificates/{name}`. |
| `request_id` | string | AppViewX workflow request id (populated when available). |
| `connector_id` | string | AppViewX application-connector id. |
| `status_code` | int | HTTP status code of the push request. |
| `success` | bool | `true` when the push completed successfully. |

## Lifecycle Notes

- **Destroy** removes only the Terraform state; the certificate is **not** deleted from GCP Certificate Manager on destroy.
- Use `appviewx_create_certificate.revoke_on_destroy` to revoke the certificate in AppViewX on destroy.
- Use `appviewx_create_certificate.wait_for_issuance` to ensure the certificate is fully issued before the push runs.
- `selected_profiles` accepts only the profiles configured in AppViewX for the GCP connector. Contact your AppViewX administrator for the correct profile IDs.
