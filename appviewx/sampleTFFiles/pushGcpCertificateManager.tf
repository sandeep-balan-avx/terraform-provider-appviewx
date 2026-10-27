terraform {
  required_providers {
    appviewx = {
      source = "appviewx.com/provider/appviewx"
    }
  }
}

provider "appviewx" {
  appviewx_environment_ip       = var.appviewx_ip
  appviewx_environment_port     = var.appviewx_port
  appviewx_environment_is_https = true
  # Auth via APPVIEWX_TERRAFORM_CLIENT_ID / APPVIEWX_TERRAFORM_CLIENT_SECRET env vars
}

# ---------------------------------------------------------------------------
# 1. Create (or renew) the certificate in AppViewX
# ---------------------------------------------------------------------------
resource "appviewx_create_certificate" "this" {
  common_name            = "dev.api.example.com"
  dns_names              = ["dev.api.example.com", "usc1.dev.api.example.com"]
  hash_function          = "SHA384"
  key_type               = "RSA"
  bit_length             = "4096"
  certificate_authority  = "Sectigo"
  ca_setting_name        = "Example CA Setting"
  certificate_type       = "External Certificates - customer facing or internet facing (public root)"
  certificate_group_name = "Application-Load-Balancers-GCP"
  validity_days          = 7
  validity_unit          = "days"
  validity_unit_value    = 7

  # Feature 5 – Auto-renewal / regeneration
  is_auto_renewal         = true
  renew_before            = "30"
  auto_regenerate_enabled = true

  # Feature 3 – Block until the cert is issued so the push runs against a ready cert
  wait_for_issuance              = true
  issuance_timeout_seconds       = 300
  issuance_poll_interval_seconds = 10

  # Feature 6 – Revoke on destroy
  revoke_on_destroy = true
  revoke_reason     = "Cessation of operation"
}

# ---------------------------------------------------------------------------
# 2a. Push to GCP Certificate Manager (push-only, no Load Balancer binding)
# ---------------------------------------------------------------------------
resource "appviewx_push_gcp_certificate_manager" "push_only" {
  certificate_id     = appviewx_create_certificate.this.resource_id
  certificate_name   = "dev-api-example-com"
  project            = var.project_id
  location           = var.region
  gcp_connector_name = "GCP connector"
  selected_profiles  = ["GCP.Single.All:${var.project_id}:Certificate Manager"]
  is_new_certificate = true
  push_automatically = true

  # Feature 4 – Wait for push completion
  wait_for_completion  = true
  wait_timeout_seconds = 600
  poll_interval_seconds = 10
}

output "gcp_certificate_id_push_only" {
  value = appviewx_push_gcp_certificate_manager.push_only.certificate_manager_certificate_id
}

# ---------------------------------------------------------------------------
# 2b. Push to GCP Certificate Manager AND bind to a Load Balancer via cert map
#     (use a "Load Balancer" type profile in AppViewX)
# ---------------------------------------------------------------------------
resource "appviewx_push_gcp_certificate_manager" "push_and_bind" {
  certificate_id     = appviewx_create_certificate.this.resource_id
  certificate_name   = "dev-api-example-com"
  project            = var.project_id
  location           = var.region
  gcp_connector_name = "GCP connector"
  selected_profiles  = ["GCP.Single.All:${var.project_id}:Load Balancer"]
  is_new_certificate = true
  push_automatically = true

  # Feature 2 – Load Balancer binding via Certificate Map
  certificate_map_name             = "dev-api-cert-map"
  is_new_certificate_map           = true
  certificate_map_entry_name       = "dev-api-cert-map-entry"
  is_new_certificate_map_entry     = true
  is_primary_certificate_map_entry = false   # set to true for a PRIMARY matcher
  hostname                         = "dev.api.example.com"

  # Feature 4 – Wait for push completion
  wait_for_completion   = true
  wait_timeout_seconds  = 600
  poll_interval_seconds = 10
}

output "gcp_certificate_id_lb" {
  value = appviewx_push_gcp_certificate_manager.push_and_bind.certificate_manager_certificate_id
}
