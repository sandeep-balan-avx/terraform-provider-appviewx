# Certificate Creation

The `appviewx_create_certificate` resource follows a structured flow to create and manage certificates using the AppViewX platform.

## Process Overview

1. **Input Parameters**:
   - The resource accepts various input parameters such as `common_name`, `organization`, `organizational_unit`, `country`, `state`, `locality`, and more. These parameters define the attributes of the certificate to be created. All the parameters are describe below.

2. **Certificate Creation**:
   - When the resource is applied, the certificate creation process is initiated. The resource sends a request to the AppViewX API with the provided parameters to generate the certificate.

3. **Synchronous vs Asynchronous Mode**:
   - If `is_sync` is set to `true`, the certificate is created in synchronous mode, and the certificate is downloaded immediately after creation.
   - If `is_sync` is set to `false`, the certificate is created in asynchronous mode, and the resource ID is returned for later use in downloading the certificate. Users can also use `common_name` and `serial_number` for downloading the certificate later.

4. **Certificate Download Options**:
   - By default, certificates and private keys are downloaded to specified file paths (`certificate_download_path`, `key_download_path`).
   - Alternatively, when `store_certificate_in_state` is enabled, the certificate and private key content are stored directly in the Terraform state instead of being written to the filesystem, providing better integration with other Terraform resources.

5. **Custom and Vendor-Specific Fields**:
   - The resource supports adding custom fields (`custom_fields`) and vendor-specific fields (`vendor_specific_fields`) to include additional metadata or configurations required by the Certificate Authority.

6. **Certificate Grouping**:
   - Certificates can be grouped using the `certificate_group_name` attribute, where the certificates will be placed in AppViewX Certificate Inventory.

7. **Certificate Download**:
   - The certificate can be downloaded to a specified path using the `certificate_download_path` attribute. The format of the downloaded certificate is defined by `certificate_download_format`.
   - If `certificate_chain_required` is set to `true`, the certificate chain is included in the download.

8. **Private Key Download**:
   - If `key_download_path` is specified, the private key associated with the certificate is downloaded to the given path. The key can be password-protected using the `key_download_password` attribute.



## Attributes

The `appviewx_create_certificate` resource supports the following attributes:

### Required Attributes

- **`common_name`** (string): The domain name or identifier for the certificate.
- **`organization`** (string): The name of the organization requesting the certificate.
- **`organizational_unit`** (string): The organizational unit or department.
- **`country`** (string): The two-letter country code.
- **`state`** (string): The state or province name.
- **`locality`** (string): The city or locality name.
- **`validity_unit`** (string): The unit of validity for the certificate. Possible values are years, days, months
- **`validity_unit_value`** (integer): The value for the validity unit.
- **`key_algorithm`** (string): The cryptographic algorithm for the key. Possible values are RSA, DSA and EC
- **`key_size`** (integer): The size of the key in bits. Possible values are 1024, 2048, 3072, 4096, 7680, 8192.
- **`certificate_authority`** (string): The name of the Certificate Authority (CA) to issue the certificate. Currently supported CA via AppViewX Terraform is **AppViewX CA**, **Microsoft Enterprise**, **Microsoft Standalone** and **Comodo Certificate Manager** (Sectigo).
- **`ca_setting_name`** (string): The specific CA setting to use for certificate issuance which has been configured in the AppViewX Certificate Authority GUI.

### Optional Attributes

- **`certificate_type`** (string): The type of certificate based on the Certificate Authority.
- **`dns_names`** (list of strings): Additional DNS names to include in the certificate.
- **`custom_fields`** (map): Custom fields to include in the certificate request based on the certificate authority.
- **`vendor_specific_fields`** (map): Vendor-specific fields for additional configurations based on the certificate authority.
- **`certificate_group_name`** (string): The name of the group to which the certificate belongs in AppViewX, deafults to **Default** certificate group in AppViewX.
- **`is_sync`** (boolean): Whether the certificate creation is synchronous (`true`) or asynchronous (`false`).

- **`certificate_download_path`** (string): The file path to download the certificate.
- **`certificate_download_format`** (string): The format of the downloaded certificate. Possible values are PEM, CER, CRT, DER, P12, PFX
- **`certificate_download_password`** (string): The password for the downloaded certificate file.
- **`certificate_chain_required`** (boolean): Whether to include the certificate chain in the downloaded certificate.

- **`key_download_path`** (string): The file path to download the private key seperately.
- **`key_download_password`** (string): The password for the downloaded private key. This is required to download the private key from AppViewX and by default the key is password protected from AppViewX.
- **`download_password_protected_key`** (boolean): To specify whether the private key should be downloaded as password-protected or plain private key. If this is enabled then the password protected key is downloaded as such, but if this is disabled then the password protected key is decrypted using the provided password using openssl and saved in the specified path automatically.
- **`store_certificate_in_state`** (boolean): When enabled, stores the certificate and private key content directly in the Terraform state instead of writing them to the filesystem. Default: `false`. **This option only works in synchronous mode (`is_sync = true`)** because in asynchronous mode, the certificate has not yet been issued and there is nothing to download. In async mode, use the `appviewx_download_certificate` resource later after the certificate is issued.

> **Note**: Private key download is optional and can be ignored if the certificate download format specified is P12 or PFX.

### Computed Attributes

The following attributes are populated only when **`store_certificate_in_state = true` AND `is_sync = true`** (synchronous mode with state storage enabled):

- **`certificate_content`** (string): The complete PEM-formatted certificate content, including the `BEGIN CERTIFICATE` and `END CERTIFICATE` headers.
- **`key_content`** (string, sensitive): The complete PEM-formatted private key content. This attribute is marked as **sensitive** and is hidden from Terraform plan and apply output.


## Example Usage

### Certificate Creation In Synchronous Mode with File-Based Download
```hcl
resource "appviewx_create_certificate" "createcert" {
   common_name                = "sample.example.com"
   hash_function              = "SHA256"
   key_type                   = "RSA"
   bit_length                 = "2048"
   certificate_authority      = "Certificate Authority"
   ca_setting_name            = "CA Setting Name"
   certificate_type           = "EliteSSL Certificate"
   dns_names                  = ["example.com", "example123.com"]
   custom_fields              = { "field_name1" = "value1", "field_name2" = "value2" }
   vendor_specific_fields     = { "field_name" = "value", "field_name2" = "value2" }
   validity_unit              = "years"
   validity_unit_value        = 2
   certificate_group_name     = "AppViewX Certificate Group Name"
   certificate_download_path  = "/path/to/directory"
   certificate_download_format = "PEM"
   certificate_chain_required = true
   key_download_path          = "/path/to/directory"
   key_download_password      = "password"
   download_password_protected_key = false
   is_sync                    = true
}
```

### Certificate Creation In Synchronous Mode with State Storage
```hcl
resource "appviewx_create_certificate" "createcert" {
   common_name                = "sample.example.com"
   hash_function              = "SHA256"
   key_type                   = "RSA"
   bit_length                 = "2048"
   certificate_authority      = "Certificate Authority"
   ca_setting_name            = "CA Setting Name"
   certificate_type           = "EliteSSL Certificate"
   dns_names                  = ["example.com", "example123.com"]
   custom_fields              = { "field_name1" = "value1", "field_name2" = "value2" }
   vendor_specific_fields     = { "field_name" = "value", "field_name2" = "value2" }
   validity_unit              = "years"
   validity_unit_value        = 2
   certificate_group_name     = "AppViewX Certificate Group Name"
   certificate_download_format = "PEM"
   certificate_chain_required = true
   key_download_password      = "password"
   store_certificate_in_state = true
   is_sync                    = true
}
```

### Certificate Creation In Asynchronous Mode
```hcl
resource "appviewx_create_certificate" "createcert" {
   common_name                = "sampe.example.com"
   hash_function              = "SHA256"
   key_type                   = "RSA"
   bit_length                 = "2048"
   certificate_authority      = "Certificate Authority"
   ca_setting_name            = "Appviewx CA Setting Name"
   certificate_type           = "EliteSSL Certificate"
   dns_names                  = ["example.com", "example123.com"]
   custom_fields              = { "field_name" = "value" }
   vendor_specific_fields     = { "field_name" = "value" }
   validity_unit              = "years"
   validity_unit_value        = 2
   certificate_group_name     = "AppViewX Certificate Group Name"
   is_sync                    = false
}

resource "time_sleep" "wait" {
   depends_on      = [appviewx_create_certificate.createcert]
   create_duration = "10s" // This is configurable based on the usage
}

resource "appviewx_download_certificate" "downloadcert" {
   depends_on                  = [time_sleep.wait]
   resource_id                 = appviewx_create_certificate.createcert.resource_id
   certificate_download_path   = "/path/to/directory"
   certificate_download_format = "P12"
   certificate_download_password = "password"
   certificate_chain_required  = true
   key_download_path           = "/path/to/directory"
   key_download_password       = "password"
   download_password_protected_key = false
}
```

> **NOTE** Here the `resource_id` field will be propagated in the terraform functionality in the backend so it might not be seen anywhere in the input field in create certificate resource.

> **NOTE** In Asynchronous mode the certificate issuance status will not be checked, and there are no rety mechanisms available.

> **NOTE** When `store_certificate_in_state = true`, no certificate or private key files are created on the local filesystem. The certificate content is stored in the Terraform state, and `key_content` is treated as a sensitive attribute.

## Import

To import an existing certificate into the Terraform state, use the following command:

```bash
terraform import appviewx_create_certificate.createcert <resource_id>
```
Replace `<resource_id>` with the actual resource ID of the certificate you want to import.

## Lifecycle & Updates

An issued certificate cannot be mutated in place — the attributes that define it are fixed at issuance. Every attribute that shapes the certificate (or the files written at create time) is therefore **force-new**: changing any of them causes Terraform to **destroy the old certificate resource and create a new one** (i.e. issue a fresh certificate).

Force-new attributes include: `common_name`, `dns_names`, `hash_function`, `key_type`, `bit_length`, `custom_fields`, `vendor_specific_fields`, `certificate_authority`, `certificate_group_name`, `ca_setting_name`, `certificate_type`, `validity_days`, `validity_unit`, `validity_unit_value`, `is_sync`, all `*_download_*` / `certificate_chain_required` attributes, `store_certificate_in_state`, the auto-renewal attributes, and the wait-for-issuance attributes.

The only attributes that update **in place** (without re-issuing the certificate) are the destroy-time revocation knobs, since they only take effect on `terraform destroy`: `revoke_on_destroy`, `revoke_reason`, `revoke_comments`.

> **NOTE** If `revoke_on_destroy = true`, a force-new change revokes the existing certificate as part of the destroy step before the replacement is issued.

### Auto-renewal (optional)

| Name | Default | Description |
|---|---|---|
| `is_auto_renewal` | `false` | Enable automatic renewal in AppViewX. |
| `renew_before` | | Days before expiry to renew (e.g. `"30"`). Only sent when `is_auto_renewal = true`. |
| `auto_regenerate_enabled` | `false` | Regenerate a new key/CSR on renewal (vs. renew with the existing key). Only sent when `is_auto_renewal = true`. |

### Wait for issuance (optional)

| Name | Default | Description |
|---|---|---|
| `wait_for_issuance` | `false` | Block until the certificate is issued (polls the create request to completion) before returning. Certificate creation is asynchronous, so **enable this when a dependent resource pushes the certificate in the same apply** (e.g. `appviewx_push_gcp_certificate_manager`) — otherwise the push can race ahead of issuance. |
| `issuance_timeout_seconds` | `600` | Max seconds to wait for issuance. Must be >= 1. |
| `issuance_poll_interval_seconds` | `10` | Seconds between issuance status polls. Must be >= 1. |

### Revoke on destroy (optional)

| Name | Default | Description |
|---|---|---|
| `revoke_on_destroy` | `false` | Revoke the certificate in AppViewX on `terraform destroy`. |
| `revoke_reason` | `Cessation of operation` | One of: Unspecified, Key compromise, CA compromise, Affiliation Changed, Superseded, Cessation of operation. |
| `revoke_comments` | | Optional free-text comment sent with the revocation. |
