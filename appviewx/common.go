package appviewx

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"terraform-provider-appviewx/appviewx/config"
	"terraform-provider-appviewx/appviewx/constants"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func printRequest(types, url string, headers map[string]interface{}, requestBody []byte) {
	log.Println("[INFO] ***************** Making a API Request to AppViewX **********************")
	log.Println("[INFO] HTTP Method : ", types)
	log.Println("[INFO] URL : ", url)
	//log.Println("[DEBUG] Headers : ", headers)
	//log.Println("[DEBUG] Request Payload : ", string(requestBody))
	log.Println("[INFO] *********************************************************")
}

// TODO: cleanup to be done
func GetSession(
	appviewxUserName,
	appviewxPassword,
	appviewxEnvironmentIP,
	appviewxEnvironmentPort,
	appviewxGwSource string,
	appviewxEnvironmentIsHTTPS bool,
) (output string, err error) {

	log.Println("[INFO] Request received for fetching session id")

	payload := make(map[string]interface{})

	headers := make(map[string]interface{})
	headers[constants.CONTENT_TYPE] = constants.APPLICATION_JSON
	headers[constants.ACCEPT] = constants.APPLICATION_JSON
	headers[constants.USERNAME] = appviewxUserName
	headers[constants.PASSWORD] = appviewxPassword

	actionID := constants.APPVIEWX_ACTION_ID_LOGIN

	queryParams := make(map[string]string)
	queryParams[constants.GW_SOURCE] = appviewxGwSource

	url := GetURL(appviewxEnvironmentIP, appviewxEnvironmentPort, actionID, queryParams, appviewxEnvironmentIsHTTPS)

	payloadContents, err := json.Marshal(payload)
	if err != nil {
		log.Println("[ERROR] Error in marshalling the payload", payload, err)
		return "", err
	}

	payloadContentsReader := bytes.NewReader(payloadContents)

	printRequest(constants.POST, url, headers, payloadContents)

	client := &http.Client{Transport: HTTPTransport()}
	req, err := http.NewRequest(constants.POST, url, payloadContentsReader)
	if err != nil {
		log.Println("[ERROR] Error in creating the new reqeust", err)
		return "", err
	}

	for key, value := range headers {
		value1 := fmt.Sprintf("%v", value)
		key1 := fmt.Sprintf("%v", key)
		req.Header.Add(key1, value1)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Println("[ERROR] Error in executing the request", err)
		return "", err
	}
	log.Println("[INFO] Response status code : ", resp.Status)
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		responseBody, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			log.Println("[ERROR] Response obtained : ", string(responseBody))
			return "", errors.New("error in getting the session id due to " + string(responseBody))
		}
		return "", errors.New("error in getting the session id, status: " + resp.Status)
	}
	responseContents, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("[ERROR] error in reading the response body", err)
		return "", err
	}

	map1 := make(map[string]interface{})
	err = json.Unmarshal(responseContents, &map1)
	if err != nil {
		log.Println("[ERROR] Error in Unmarshalling the responseContents", err)
		return "", err
	}

	if map1[constants.RESPONSE] != nil {
		responseMap := map1[constants.RESPONSE].(map[string]interface{})
		if responseMap != nil && responseMap[constants.SESSION_ID] != nil {
			log.Println("[INFO] Session id retrieval success, sessionid will be used for AppViewX API calls")
			return responseMap[constants.SESSION_ID].(string), nil
		}
	}
	log.Println("[ERROR] Session id retrieval failed ")
	return "", nil
}

func HTTPTransport() *http.Transport {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return tr
}

func GetDownloadFormat(resourceData *schema.ResourceData) string {
	if resourceData.Get(constants.CERTIFICATE_DOWNLOAD_FORMAT) != nil {
		return resourceData.Get(constants.CERTIFICATE_DOWNLOAD_FORMAT).(string)
	} else {
		return "CRT"
	}
}

func GetDownloadFilePath(resourceData *schema.ResourceData, commonName, downloadFormat string) (string, error) {
	workingDir, _ := os.Getwd()
	if resourceData.Get(constants.CERTIFICATE_DOWNLOAD_PATH) == nil && resourceData.Get(constants.COMMON_NAME) != nil {
		log.Println("[INFO] " + "Download path not provided hence saving file in current working directory with common name")
		return workingDir + commonName + "." + strings.ToLower(downloadFormat), nil
	} else {
		downloadPath := resourceData.Get(constants.CERTIFICATE_DOWNLOAD_PATH).(string)
		log.Println("[INFO] Download path provided = ", downloadPath)

		// Validate that the target directory exists and is accessible
		parentDir := filepath.Dir(downloadPath)
		if parentDir == "" || parentDir == "." {
			parentDir = workingDir
		}

		fileInfo, err := os.Stat(parentDir)
		if err != nil {
			if os.IsNotExist(err) {
				log.Println("[ERROR] Invalid certificate download path: directory does not exist: " + parentDir)
				log.Println("[ERROR] Cannot write to path: " + downloadPath)
				log.Println("[ERROR] Check file permissions and directory access")
				return "", errors.New("invalid certificate download path: directory does not exist: " + parentDir)
			}
			log.Println("[ERROR] Cannot access directory: " + parentDir + " (error: " + err.Error() + ")")
			return "", errors.New("cannot access directory: " + parentDir + " (error: " + err.Error() + ")")
		}

		if !fileInfo.IsDir() {
			log.Println("[ERROR] Path is not a directory: " + parentDir)
			return "", errors.New("path is not a directory: " + parentDir)
		}

		// Path exists and is a valid directory - return full file path
		fullPath := filepath.Join(parentDir, commonName+"."+strings.ToLower(downloadFormat))
		log.Println("[INFO] Certificate will be saved to: " + fullPath)
		return fullPath, nil
	}
}

func GetDownloadFilePathForKey(resourceData *schema.ResourceData, commonName, downloadFormat string) (string, error) {
	workingDir, _ := os.Getwd()
	if resourceData.Get(constants.KEY_DOWNLOAD_PATH) == nil && resourceData.Get(constants.COMMON_NAME) != nil {
		log.Println("[INFO] " + "Download path not provided hence saving file in current working directory with common name")
		return workingDir + commonName + "." + strings.ToLower(downloadFormat), nil
	} else {
		downloadPath := resourceData.Get(constants.KEY_DOWNLOAD_PATH).(string)
		log.Println("[INFO] Download path provided = ", downloadPath)

		// Validate that the target directory exists and is accessible
		parentDir := filepath.Dir(downloadPath)
		if parentDir == "" || parentDir == "." {
			parentDir = workingDir
		}

		fileInfo, err := os.Stat(parentDir)
		if err != nil {
			if os.IsNotExist(err) {
				log.Println("[ERROR] Invalid key download path: directory does not exist: " + parentDir)
				log.Println("[ERROR] Cannot write to path: " + downloadPath)
				log.Println("[ERROR] Check file permissions and directory access")
				return "", errors.New("invalid key download path: directory does not exist: " + parentDir)
			}
			log.Println("[ERROR] Cannot access directory: " + parentDir + " (error: " + err.Error() + ")")
			return "", errors.New("cannot access directory: " + parentDir + " (error: " + err.Error() + ")")
		}

		if !fileInfo.IsDir() {
			log.Println("[ERROR] Path is not a directory: " + parentDir)
			return "", errors.New("path is not a directory: " + parentDir)
		}

		// Path exists and is a valid directory - return full file path
		fullPath := filepath.Join(parentDir, commonName+"."+strings.ToLower(downloadFormat))
		log.Println("[INFO] Private key will be saved to: " + fullPath)
		return fullPath, nil
	}
}

func GetDownloadPassword(resourceData *schema.ResourceData, downloadFormat string, configAppviewxEnvironment *config.AppViewXEnvironment) (string, bool) {
	password := getPasswordWithPriority(configAppviewxEnvironment.ProviderCertDownloadPassword, resourceData.Get(constants.CERTIFICATE_DOWNLOAD_PASSWORD).(string))
	if password != "" && (downloadFormat == "PFX" || downloadFormat == "JKS" || downloadFormat == "P12") {
		return password, true
	} else if password == "" && (downloadFormat == "PFX" || downloadFormat == "JKS" || downloadFormat == "P12") {
		log.Println("[ERROR] Password not found for the specified download format - " + downloadFormat)
		return "", false
	}
	return "", true
}

// getPasswordWithPriority returns password with provider priority over resource
func getPasswordWithPriority(providerPassword, resourcePassword string) string {
	if providerPassword != "" {
		log.Println("[INFO] Using password from provider configuration")
		return providerPassword
	}
	if resourcePassword != "" {
		log.Println("[INFO] Using password from resource configuration")
		return resourcePassword
	}
	log.Println("[INFO] No password provided at provider or resource level")
	return ""
}

// validateAndGetDirectoryPath checks if the directory for the given file path is accessible.
// Returns the directory path if valid, otherwise returns error details.
func validateAndGetDirectoryPath(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path is empty")
	}
	dir := filepath.Dir(filePath)
	if dir == "" || dir == "." {
		dir = "."
	}
	fileInfo, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("directory does not exist: %s", dir)
		}
		return "", fmt.Errorf("cannot access directory: %s (error: %v)", dir, err)
	}
	if !fileInfo.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", dir)
	}
	return dir, nil
}

func downloadCertificateFromAppviewx(appviewxResourceId, commonName, serialNumber, downloadFormat, downloadPassword, downloadPath string, isChainRequired bool, appviewxSessionID, appviewxAccessToken string, configAppViewXEnvironment *config.AppViewXEnvironment) bool {
	// Validate file path before attempting download
	if _, err := validateAndGetDirectoryPath(downloadPath); err != nil {
		log.Println("[ERROR] Invalid certificate download path: " + err.Error())
		log.Println("[ERROR] Certificate file cannot be written to: " + downloadPath)
		return false
	}

	httpMethod := config.HTTPMethodPost
	appviewxEnvironmentIP := configAppViewXEnvironment.AppViewXEnvironmentIP
	appviewxEnvironmentPort := configAppViewXEnvironment.AppViewXEnvironmentPort
	appviewxEnvironmentIsHTTPS := configAppViewXEnvironment.AppViewXIsHTTPS
	headers := frameHeaders()
	url := GetURL(appviewxEnvironmentIP, appviewxEnvironmentPort, config.DownloadCertificateActionId, frameQueryParams(), appviewxEnvironmentIsHTTPS)
	payload := frameDownloadCertificatePayload(appviewxResourceId, commonName, serialNumber, downloadFormat, downloadPassword, isChainRequired)
	requestBody, err := json.Marshal(payload)
	if err != nil {
		log.Println("[ERROR] error in Marshalling the payload ", payload, err)
		return false
	}
	client := &http.Client{Transport: HTTPTransport()}

	printRequest(httpMethod, url, headers, requestBody)

	req, err := http.NewRequest(httpMethod, url, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Println("[ERROR] error in creating new Request", err)
		return false
	}

	for key, value := range headers {
		value1 := fmt.Sprintf("%v", value)
		key1 := fmt.Sprintf("%v", key)
		req.Header.Add(key1, value1)
	}
	if appviewxSessionID != "" {
		req.Header.Add(constants.SESSION_ID, appviewxSessionID)
	} else {
		req.Header.Add(constants.TOKEN, appviewxAccessToken)
	}
	httpResponse, err := client.Do(req)
	if err != nil {
		log.Println("[ERROR] error in http request", err)
		return false
	} else {
		log.Println("[INFO] Request for downloading the certificate submitted successfully")
	}
	log.Println("[INFO] Response status code : ", httpResponse.Status)
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		responseBody, err := io.ReadAll(httpResponse.Body)
		if err == nil {
			log.Println("[ERROR] Response obtained : ", string(responseBody))
			return false
		}
	}
	responseByte, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		log.Println("[ERROR] ", err)
		return false
	} else {
		err = os.WriteFile(downloadPath, responseByte, 0600)
		if err != nil {
			log.Println("[ERROR] Failed to write certificate file: " + err.Error())
			log.Println("[ERROR] Cannot write to path: " + downloadPath)
			log.Println("[ERROR] Check file permissions and directory access")
			return false
		} else {
			log.Println("[INFO] Downloaded certificate file and available in ", downloadPath)
			return true
		}
	}

}

func downloadKey(resourceData *schema.ResourceData, resourceID, appviewxSessionID, accessToken string, configAppViewXEnvironment *config.AppViewXEnvironment, storeInState bool) error {
	commonName := resourceData.Get(constants.COMMON_NAME).(string)

	// Only compute download path if we're writing to files
	var downloadPath string
	var err error
	if !storeInState {
		downloadPath, err = GetDownloadFilePathForKey(resourceData, commonName+"_key", "PEM")
		if err != nil {
			log.Println("[ERROR] Failed to validate key download path: " + err.Error())
			return err
		}
	}

	providerKeyPassword := configAppViewXEnvironment.ProviderKeyDownloadPassword
	resourceKeyPassword := resourceData.Get(constants.KEY_DOWNLOAD_PASSWORD).(string)
	downloadPassword := getPasswordWithPriority(providerKeyPassword, resourceKeyPassword)
	downloadPasswordProtectedKey := resourceData.Get(constants.DOWNLOAD_PASSWORD_PROTECTED_KEY).(bool)

	if downloadPassword == "" {
		log.Println("[ERROR] Password not found for private key download")
		return errors.New("[ERROR] Password not found for private key download")
	}

	searchResponse := searchCertificate(resourceID, appviewxSessionID, accessToken, configAppViewXEnvironment)
	objects := searchResponse.AppviewxResponse.ResponseObject.Objects
	if len(objects) == 0 || objects[0].UUID == "" {
		log.Println("[ERROR] Cannot find the UUID for the resource id " + resourceID + " to proceed with key download")
		return errors.New("[ERROR] Certificate details was not found to download the private key")
	}
	uuid := objects[0].UUID
	log.Println("[INFO] UUID for the resource id " + resourceID + " was obtained successfully")

	// Handle key download based on storeInState flag
	if storeInState {
		// Fetch key content and store in state
		keyContent, success, errMsg := downloadKeyContentFromAppviewx(uuid, downloadPassword, downloadPasswordProtectedKey, appviewxSessionID, accessToken, configAppViewXEnvironment)
		if !success {
			log.Println("[ERROR] Private key download failed: " + errMsg)
			return errors.New("[ERROR] Private key was not downloaded: " + errMsg)
		}
		resourceData.Set(constants.KEY_CONTENT, keyContent)
		log.Println("[INFO] Private key content stored in Terraform state")
	} else {
		// Use existing file-based behavior
		if downloadSuccess := downloadKeyFromAppviewx(uuid, downloadPassword, downloadPath, downloadPasswordProtectedKey, appviewxSessionID, accessToken, configAppViewXEnvironment); downloadSuccess {
			log.Println("[INFO] Private key downloaded successfully in the specified path")
		} else {
			log.Println("[ERROR] Private key was not downloaded in the specified path")
			return errors.New("[ERROR] Private key was not downloaded in the specified path")
		}
	}
	return nil
}

func downloadKeyFromAppviewx(uuid, downloadPassword, downloadPath string, downloadPasswordProtectedKey bool, appviewxSessionID, appviewxAccessToken string, configAppViewXEnvironment *config.AppViewXEnvironment) bool {
	// Validate file path before attempting download
	if _, err := validateAndGetDirectoryPath(downloadPath); err != nil {
		log.Println("[ERROR] Invalid key download path: " + err.Error())
		log.Println("[ERROR] Key file cannot be written to: " + downloadPath)
		return false
	}

	httpMethod := config.HTTPMethodPost
	var response config.AppviewxDownloadKeyResponse
	var responseByte []byte
	appviewxEnvironmentIP := configAppViewXEnvironment.AppViewXEnvironmentIP
	appviewxEnvironmentPort := configAppViewXEnvironment.AppViewXEnvironmentPort
	appviewxEnvironmentIsHTTPS := configAppViewXEnvironment.AppViewXIsHTTPS
	headers := frameHeaders()
	url := GetURL(appviewxEnvironmentIP, appviewxEnvironmentPort, config.DownloadKeyActionId, frameQueryParams(), appviewxEnvironmentIsHTTPS)
	payload := frameDownloadKeyPayload(uuid, downloadPassword)
	requestBody, err := json.Marshal(payload)
	if err != nil {
		log.Println("[ERROR] error in Marshalling the payload ", payload, err)
		return false
	}
	client := &http.Client{Transport: HTTPTransport()}

	printRequest(httpMethod, url, headers, requestBody)

	req, err := http.NewRequest(httpMethod, url, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Println("[ERROR] error in creating new Request", err)
		return false
	}

	for key, value := range headers {
		value1 := fmt.Sprintf("%v", value)
		key1 := fmt.Sprintf("%v", key)
		req.Header.Add(key1, value1)
	}
	if appviewxSessionID != "" {
		req.Header.Add(constants.SESSION_ID, appviewxSessionID)
	} else {
		req.Header.Add(constants.TOKEN, appviewxAccessToken)
	}
	httpResponse, err := client.Do(req)
	if err != nil {
		log.Println("[ERROR] error in http request", err)
		return false
	} else {
		log.Println("[INFO] Request for downloading the private submitted successfully")
	}
	log.Println("[INFO] Response status code : ", httpResponse.Status)
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		responseBody, err := io.ReadAll(httpResponse.Body)
		if err == nil {
			log.Println("[ERROR] Response obtained : ", string(responseBody))
			return false
		}
	}
	if responseByte, err = io.ReadAll(httpResponse.Body); err != nil {
		log.Println("[ERROR] Error while obtaining the response due to : ", err)
		return false
	}
	if err = json.Unmarshal(responseByte, &response); err != nil {
		log.Println("[ERROR] Error while obtaining the response due to : ", err)
		return false
	} else if response.AppviewxResponse.Status == "Success" {
		if downloadPasswordProtectedKey {
			log.Println("[INFO] Downloading the password protected private key file content. Kindly use the password provided in the .tf file")
			if err := writeKeyToFile(downloadPath, []byte(response.AppviewxResponse.PrivateKey)); err != nil {
				return false
			}
		} else {
			if err := decryptPasswordProtectedKeyAndDownloadKey(response.AppviewxResponse.PrivateKey, downloadPassword, downloadPath); err != nil {
				return false
			}
		}
	} else {
		log.Println("[ERROR] Error while obtaining the response due to : ", err)
		return false
	}
	return true
}

func decryptPasswordProtectedKeyAndDownloadKey(encryptedPrivateKey, password string, downloadPath string) error {
	// Validate output path before attempting to write
	if _, err := validateAndGetDirectoryPath(downloadPath); err != nil {
		log.Println("[ERROR] Invalid key output path: " + err.Error())
		log.Println("[ERROR] Cannot write decrypted key to: " + downloadPath)
		return errors.New("invalid key download path: " + err.Error())
	}

	log.Println("[INFO] Decrypting the password protected private key file content")
	var file *os.File
	var err error
	file, err = os.CreateTemp("", "appviewx_enc_key_*.pem")
	if err != nil {
		log.Println("[ERROR] Error creating temp file: " + err.Error())
		return errors.New("cannot create temporary file for key decryption: " + err.Error())
	}
	tempFile := file.Name()
	defer os.Remove(tempFile)

	_, err = file.WriteString(encryptedPrivateKey)
	file.Close() // flush before openssl reads it
	if err != nil {
		log.Println("[ERROR] Error writing to temp file: " + err.Error())
		return errors.New("error preparing key for decryption: " + err.Error())
	}
	cmd := exec.Command("openssl", "pkey", "-in", tempFile, "-out", downloadPath, "-passin", "pass:"+password)

	err = cmd.Run()
	if err != nil {
		log.Printf("[ERROR] Error executing OpenSSL command: %v\n", err)
		log.Println("[ERROR] Failed to decrypt private key. Ensure OpenSSL is installed and password is correct.")
		return errors.New("error while decrypting the private key: " + err.Error())
	}
	log.Println("[INFO] Private key decrypted successfully and saved in the specified path")
	return nil
}

func writeKeyToFile(downloadPath string, fileContent []byte) error {
	// Validate path before attempting to write
	if _, err := validateAndGetDirectoryPath(downloadPath); err != nil {
		log.Println("[ERROR] Invalid key file path: " + err.Error())
		log.Println("[ERROR] Cannot write key to: " + downloadPath)
		return errors.New("invalid key file path: " + err.Error())
	}

	if err := os.WriteFile(downloadPath, fileContent, 0600); err != nil {
		log.Println("[ERROR] Failed to write key file: " + err.Error())
		log.Println("[ERROR] Cannot write to path: " + downloadPath)
		log.Println("[ERROR] Check file permissions and directory access")
		return errors.New("failed to write key file: " + err.Error())
	}
	log.Println("[INFO] Downloaded private key file and available in ", downloadPath)
	return nil
}

func searchCertificate(resourceID, appviewxSessionID, accessToken string, configAppViewXEnvironment *config.AppViewXEnvironment) config.AppviewxSearchCertResponse {
	var response config.AppviewxSearchCertResponse
	httpMethod := config.HTTPMethodPost
	appviewxEnvironmentIP := configAppViewXEnvironment.AppViewXEnvironmentIP
	appviewxEnvironmentPort := configAppViewXEnvironment.AppViewXEnvironmentPort
	appviewxEnvironmentIsHTTPS := configAppViewXEnvironment.AppViewXIsHTTPS
	headers := frameHeaders()
	url := GetURL(appviewxEnvironmentIP, appviewxEnvironmentPort, config.SearchCertificateActionId, frameQueryParams(), appviewxEnvironmentIsHTTPS)
	payload := frameSearchCertificatePayload(resourceID)
	requestBody, err := json.Marshal(payload)
	if err != nil {
		log.Println("[ERROR] error in Marshalling the payload ", payload, err)
		return response
	}
	client := &http.Client{Transport: HTTPTransport()}

	printRequest(httpMethod, url, headers, requestBody)

	req, err := http.NewRequest(httpMethod, url, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Println("[ERROR] error in creating new Request", err)
		return response
	}

	for key, value := range headers {
		value1 := fmt.Sprintf("%v", value)
		key1 := fmt.Sprintf("%v", key)
		req.Header.Add(key1, value1)
	}
	if appviewxSessionID != "" {
		req.Header.Add(constants.SESSION_ID, appviewxSessionID)
	} else {
		req.Header.Add(constants.TOKEN, accessToken)
	}
	httpResponse, err := client.Do(req)
	if err != nil {
		log.Println("[ERROR] error in http request", err)
		return response
	}
	log.Println("[INFO] Response status code : ", httpResponse.Status)
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		responseBody, err := io.ReadAll(httpResponse.Body)
		if err == nil {
			log.Println("[ERROR] Response obtained : ", string(responseBody))
			return response
		}
	}
	responseByte, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		log.Println(err)
	} else {
		err = json.Unmarshal(responseByte, &response)
		if err != nil {
			log.Println("[ERROR] Error while searching for certificate with resource id "+resourceID+" due to :", err)
		} else {
			log.Println("[INFO] Obtained response for certificate search successfully")
		}
	}
	return response
}

func frameSearchCertificatePayload(resourceId string) config.SearchCertificatePayload {
	var payload config.SearchCertificatePayload
	payload.Filter.SortOrder = "asc"
	payload.Input.ResourceId = resourceId
	return payload
}

// downloadCertificateContentFromAppviewx fetches certificate content without writing to file
// Returns (content as string, success as bool, error message if any)
func downloadCertificateContentFromAppviewx(appviewxResourceId, commonName, serialNumber, downloadFormat, downloadPassword string, isChainRequired bool, appviewxSessionID, appviewxAccessToken string, configAppViewXEnvironment *config.AppViewXEnvironment) (string, bool, string) {
	httpMethod := config.HTTPMethodPost
	appviewxEnvironmentIP := configAppViewXEnvironment.AppViewXEnvironmentIP
	appviewxEnvironmentPort := configAppViewXEnvironment.AppViewXEnvironmentPort
	appviewxEnvironmentIsHTTPS := configAppViewXEnvironment.AppViewXIsHTTPS
	headers := frameHeaders()
	url := GetURL(appviewxEnvironmentIP, appviewxEnvironmentPort, config.DownloadCertificateActionId, frameQueryParams(), appviewxEnvironmentIsHTTPS)
	payload := frameDownloadCertificatePayload(appviewxResourceId, commonName, serialNumber, downloadFormat, downloadPassword, isChainRequired)
	requestBody, err := json.Marshal(payload)
	if err != nil {
		log.Println("[ERROR] error in Marshalling the payload ", payload, err)
		return "", false, "Error marshalling payload"
	}
	client := &http.Client{Transport: HTTPTransport()}

	printRequest(httpMethod, url, headers, requestBody)

	req, err := http.NewRequest(httpMethod, url, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Println("[ERROR] error in creating new Request", err)
		return "", false, "Error creating HTTP request"
	}

	for key, value := range headers {
		value1 := fmt.Sprintf("%v", value)
		key1 := fmt.Sprintf("%v", key)
		req.Header.Add(key1, value1)
	}
	if appviewxSessionID != "" {
		req.Header.Add(constants.SESSION_ID, appviewxSessionID)
	} else {
		req.Header.Add(constants.TOKEN, appviewxAccessToken)
	}
	httpResponse, err := client.Do(req)
	if err != nil {
		log.Println("[ERROR] error in http request", err)
		return "", false, "Error making HTTP request"
	} else {
		log.Println("[INFO] Request for downloading the certificate submitted successfully")
	}
	log.Println("[INFO] Response status code : ", httpResponse.Status)
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		responseBody, err := io.ReadAll(httpResponse.Body)
		if err == nil {
			errMsg := string(responseBody)
			log.Println("[ERROR] Response obtained : ", errMsg)
			return "", false, "Error response from server"
		}
		return "", false, "Error response from server"
	}
	responseByte, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		log.Println("[ERROR] ", err)
		return "", false, "Error reading response body"
	}
	log.Println("[INFO] Downloaded certificate content successfully")
	return string(responseByte), true, ""
}

// downloadKeyContentFromAppviewx fetches private key content without writing to file
// Returns (key content as string, success as bool, error message if any)
func downloadKeyContentFromAppviewx(uuid, downloadPassword string, downloadPasswordProtectedKey bool, appviewxSessionID, appviewxAccessToken string, configAppViewXEnvironment *config.AppViewXEnvironment) (string, bool, string) {
	httpMethod := config.HTTPMethodPost
	var response config.AppviewxDownloadKeyResponse
	var responseByte []byte
	appviewxEnvironmentIP := configAppViewXEnvironment.AppViewXEnvironmentIP
	appviewxEnvironmentPort := configAppViewXEnvironment.AppViewXEnvironmentPort
	appviewxEnvironmentIsHTTPS := configAppViewXEnvironment.AppViewXIsHTTPS
	headers := frameHeaders()
	url := GetURL(appviewxEnvironmentIP, appviewxEnvironmentPort, config.DownloadKeyActionId, frameQueryParams(), appviewxEnvironmentIsHTTPS)
	payload := frameDownloadKeyPayload(uuid, downloadPassword)
	requestBody, err := json.Marshal(payload)
	if err != nil {
		log.Println("[ERROR] error in Marshalling the payload ", payload, err)
		return "", false, "Error marshalling payload"
	}
	client := &http.Client{Transport: HTTPTransport()}

	printRequest(httpMethod, url, headers, requestBody)

	req, err := http.NewRequest(httpMethod, url, bytes.NewBuffer(requestBody))
	if err != nil {
		log.Println("[ERROR] error in creating new Request", err)
		return "", false, "Error creating HTTP request"
	}

	for key, value := range headers {
		value1 := fmt.Sprintf("%v", value)
		key1 := fmt.Sprintf("%v", key)
		req.Header.Add(key1, value1)
	}
	if appviewxSessionID != "" {
		req.Header.Add(constants.SESSION_ID, appviewxSessionID)
	} else {
		req.Header.Add(constants.TOKEN, appviewxAccessToken)
	}
	httpResponse, err := client.Do(req)
	if err != nil {
		log.Println("[ERROR] error in http request", err)
		return "", false, "Error making HTTP request"
	} else {
		log.Println("[INFO] Request for downloading the private key submitted successfully")
	}
	log.Println("[INFO] Response status code : ", httpResponse.Status)
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		responseBody, err := io.ReadAll(httpResponse.Body)
		if err == nil {
			log.Println("[ERROR] Response obtained : ", string(responseBody))
			return "", false, "Error response from server"
		}
		return "", false, "Error response from server"
	}
	if responseByte, err = io.ReadAll(httpResponse.Body); err != nil {
		log.Println("[ERROR] Error while obtaining the response due to : ", err)
		return "", false, "Error reading response body"
	}
	if err = json.Unmarshal(responseByte, &response); err != nil {
		log.Println("[ERROR] Error while obtaining the response due to : ", err)
		return "", false, "Error parsing response JSON"
	} else if response.AppviewxResponse.Status == "Success" {
		if downloadPasswordProtectedKey {
			log.Println("[INFO] Using password protected private key content")
			// Return the encrypted key as-is
			return response.AppviewxResponse.PrivateKey, true, ""
		} else {
			// Decrypt the key and return the decrypted content
			decryptedKey, err := decryptPasswordProtectedKey(response.AppviewxResponse.PrivateKey, downloadPassword)
			if err != nil {
				log.Println("[ERROR] Error decrypting private key: ", err)
				return "", false, "Error decrypting private key"
			}
			return decryptedKey, true, ""
		}
	} else {
		log.Println("[ERROR] Error while obtaining the response due to : ", response.AppviewxResponse.Status)
		return "", false, "Error response from server"
	}
}

// decryptPasswordProtectedKey decrypts a password-protected key and returns the decrypted content as string
func decryptPasswordProtectedKey(encryptedPrivateKey, password string) (string, error) {
	log.Println("[INFO] Decrypting the password protected private key content")
	var file *os.File
	var err error
	file, err = os.CreateTemp("", "appviewx_enc_key_*.pem")
	if err != nil {
		log.Println("[ERROR] Error creating temp file:", err)
		return "", errors.New("error while decrypting the private key content")
	}
	tempFile := file.Name()
	defer os.Remove(tempFile)

	_, err = file.WriteString(encryptedPrivateKey)
	file.Close() // flush before openssl reads it
	if err != nil {
		log.Println("[ERROR] Error writing to temp file:", err)
		return "", errors.New("error while decrypting the private key content")
	}

	// Create a unique temporary output file for the decrypted key.
	decryptedF, err := os.CreateTemp("", "appviewx_dec_key_*.pem")
	if err != nil {
		log.Println("[ERROR] Error creating decrypted temp file:", err)
		return "", errors.New("error while decrypting the private key content")
	}
	decryptedFile := decryptedF.Name()
	decryptedF.Close()
	defer os.Remove(decryptedFile)

	cmd := exec.Command("openssl", "pkey", "-in", tempFile, "-out", decryptedFile, "-passin", "pass:"+password)

	err = cmd.Run()
	if err != nil {
		log.Printf("[ERROR] Error executing OpenSSL command: %v\n", err)
		return "", errors.New("error while decrypting the private key content")
	}

	// Read the decrypted key content
	decryptedKeyBytes, err := os.ReadFile(decryptedFile)
	if err != nil {
		log.Printf("[ERROR] Error reading decrypted key: %v\n", err)
		return "", errors.New("error while reading decrypted private key")
	}

	log.Println("[INFO] Private key decrypted successfully")
	return string(decryptedKeyBytes), nil
}

func frameDownloadCertificatePayload(appviewxResourceId, commonName, serialNumber, format, password string, isChainRequired bool) config.DownloadCertificatePayload {
	var payload config.DownloadCertificatePayload
	payload.CommonName = commonName
	payload.SerialNumber = serialNumber
	payload.Format = format
	payload.IsChainRequired = isChainRequired
	payload.Password = password
	payload.ResourceId = appviewxResourceId
	return payload
}

func frameDownloadKeyPayload(appviewxUUID, password string) config.DownloadKeyPayload {
	var payload config.DownloadKeyPayload
	payload.Password = password
	payload.UUID = appviewxUUID
	return payload
}
