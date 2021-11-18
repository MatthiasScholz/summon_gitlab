package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

var Version string

type VaultEntry struct {
	//environment_scope string
	//masked bool
	//protected bool
	Value string
	Key string
	//variable_type string
}

type VaultFile struct {
	FileName      string `json:"file_name"`
	FilePath      string `json:"file_path"`
	Size          int    `json:"size"`
	Encoding      string `json:"encoding"`
	ContentSha256 string `json:"content_sha256"`
	Ref           string `json:"ref"`
	BlobID        string `json:"blob_id"`
	CommitID      string `json:"commit_id"`
	LastCommitID  string `json:"last_commit_id"`
	Content       string `json:"content"`
}

type SecretNotFoundError struct{
	Key string
	Msg string
}

func (e *SecretNotFoundError) Error() string {
	return fmt.Sprintf("Secret not found: '%v', details: '%v'", e.Key, e.Msg)
}

func convertKey(path string) string {
	// Trim trailing "/"
	encoded := path
	if encoded[:1] == "/" {
		encoded = path[1:]
	}

	// Convert directory separator "/"
	encoded = url.PathEscape(encoded)

	// Convert possible file extension separator: "."
	encoded = strings.ReplaceAll(encoded, ".", "%2E")

	return encoded
}

func getSecretFile(url string, token string, path string) (string, error) {
	// Debugging
	// log.Printf("Getting content for %q from %q", path, url)

	// NOTE: This is needed when the Gitlab uses a certificate issue by a private CA
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{}

	// Prepare request to get file content
	path_converted := convertKey(path)
	//uri := url + path_converted + "/raw?ref=main"
	uri := url + path_converted + "?ref=main"
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		log.Printf("Error creating the request: %v", err)
		return "", err
	}
	// Debugging
	log.Printf("Requesting URL: %q", uri)
	req.Header.Add("PRIVATE-TOKEN", token)

	// Derive response
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %q", err)
		return "", err
	}
	defer resp.Body.Close()

	//We Read the response body on the line below.
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error: %q", resp.Status)
		return "", &SecretNotFoundError{path, resp.Status}
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %q", err)
		return "", err
	}
	// Debugging
	//log.Printf("Response: '%v'", string(body))

	// Parse body into struct
	var vaultFile VaultFile
	err = json.Unmarshal([]byte(body), &vaultFile)
	if err != nil {
		log.Printf("Error unmarshal response: '%v'", err)
		return "", err
	}
	// Debugging
	//log.Printf("Parsed: '%+v'", vaultFile)
	//log.Printf("File content: '%+v'", vaultFile.Content)
	return vaultFile.Content, nil
}

func getSecretValue(url string, token string, secretKey string) (string, error) {
	// Debugging
	// log.Printf("Getting %q from %q", secretKey, url)

	// NOTE: This is needed when the Gitlab uses a certificate issue by a private CA
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{}

	// Prepare request to get ALL variables
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating the request: %v", err)
		return "", err
	}
	req.Header.Add("PRIVATE-TOKEN", token)

	// Derive response
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request: %q", err)
		return "", err
	}
	defer resp.Body.Close()

	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response: %q", err)
		return "", err
	}
	// Debugging
	// log.Println("Response: ", string(body))

	// Extract secret from json response
	var secrets []VaultEntry
	err = json.Unmarshal([]byte(body), &secrets)
	if err != nil {
		log.Printf("Error unmarshal response: %q", err)
		return "", err
	}
	// Debugging
	// log.Printf("Parsed: '%+v'", secrets)

	// Search for the specific secret
	for _, s := range(secrets) {
		if s.Key == secretKey {
			return s.Value, nil
		}
	}

	log.Printf("Error secret not found for key: %q", secretKey)
	return "", &SecretNotFoundError{secretKey, "secret key not listed in the response"}
}

// Extract values from environment variables, or fatal.
func getEnv(key string) string {
	// Check environment variables
	value, ok := os.LookupEnv(key)
	if ok == true {
		return value
	}

	// Check .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %q", err)
	}
	value = os.Getenv(key)
	if value != "" {
		return value
	}

	log.Fatalln("Environment variable not set: ", key)
	return ""
}

func getToken() string {
	return getEnv("GITLAB_TOKEN")
}

// https://docs.gitlab.com/ee/api/repository_files.html#get-raw-file-from-repository
func getApiStore() string {
	return getEnv("GITLAB_VAULT_STORE")
}

func getApiVariables() string {
	return getEnv("GITLAB_VAULT_API")
}

func RetrieveSecret(key string) (string, error) {
	// Debugging
	// log.Println("Getting secret for: ", secretKey)

	token := getToken()

	// File or variable?
	var err error
	var secret string
	if strings.Contains(key, "/") == true {
		vault := getApiStore()
		secret, err = getSecretFile(vault, token, key)
	} else {
		vault := getApiVariables()
		secret, err = getSecretValue(vault, token, key)
	}

	return secret, err
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalln("A variable ID or version flag (-v, --version) must be given as the first and only argument!")
	}

	// Get the secret and key name from the argument
	argmument := os.Args[1]

	switch argmument {
	case "-v", "--version":
		fmt.Println(Version)
	default:
		secret, err := RetrieveSecret(argmument)
		if err != nil {
			// NOTE: Do not exit with error
			//       to make use of the default value support of summon.
			fmt.Fprintf(os.Stderr, "%+v", err);
			return
		}
		fmt.Println(secret)
	}
}
