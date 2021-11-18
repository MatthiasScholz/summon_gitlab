package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TODO Inject fake response by the test
func FakeVaultHandler(w http.ResponseWriter, r *http.Request) {
	responseTxt := `[
    {
        "environment_scope": "*",
        "masked": false,
        "protected": false,
        "value": "it works",
        "key": "smoketest",
        "variable_type": "env_var"
    },
    {
        "environment_scope": "*",
        "masked": false,
        "protected": false,
        "value": "email@address.de",
        "key": "user",
        "variable_type": "env_var"
    }
]
`
	fmt.Fprintf(w, responseTxt)
}

func FakeFileHandler(w http.ResponseWriter, r *http.Request) {
	response := VaultFile{Content: "file_content_here"}
	responseTxt, err := json.Marshal(&response)
	if err != nil {
		log.Fatalf("Unable to marshal JSON: '%v'", err)
	}
	fmt.Fprintf(w, string(responseTxt))
}


func TestGetSecretValueMatch(t *testing.T) {
	expect := "it works"
	fakeBackend := httptest.NewServer(http.HandlerFunc(FakeVaultHandler))
	secret, err := getSecretValue(fakeBackend.URL, "fakeToken", "smoketest")

	if err != nil {
		t.Errorf("Retrieving secret failed: %q", err)
	}
	if secret != expect {
		t.Errorf("Retrieved secret wrong: got %q want %q", secret, expect)
	}
}

func TestGetSecretValueUnMatch(t *testing.T) {
	expect := ""
	fakeBackend := httptest.NewServer(http.HandlerFunc(FakeVaultHandler))
	secret, err := getSecretValue(fakeBackend.URL, "fakeToken", "unmatched")

	if err == nil {
		t.Errorf("Retrieved a secret, got %q expected %q",secret, expect)
	}
	if secret != expect {
		t.Errorf("Retrieved secret wrong: got %q want %q", secret, expect)
	}
}

func TestGetSecretIntegration(t *testing.T) {
	secretKey := "smoketest"
	expect := "it works"

	token := getToken()
	vault := getApiVariables()
	secret, err := getSecretValue(vault, token, secretKey)

	if err != nil {
		t.Errorf("Retrieving secret failed: %q", err)
	}
	if secret != expect {
		t.Errorf("Retrieved secret wrong: got %q want %q", secret, expect)
	}
}

func TestGetSecretFile(t *testing.T) {
	secretKey := "/certificates/test.txt"
	expect := "file_content_here"

	fakeBackend := httptest.NewServer(http.HandlerFunc(FakeFileHandler))
	vault := fakeBackend.URL + "/"
	token := getToken()
	secret, err := getSecretFile(vault, token, secretKey)

	log.Printf("Received content: '%v'", secret)

	if err != nil {
		t.Errorf("Retrieving secret failed: %q", err)
	}
	if secret != expect {
		t.Errorf("Retrieved file content wrong: got %q want %q", secret, expect)
	}
}

func TestConvertKey(t *testing.T) {
	expect := "certificates%2Ftest%2Etxt"

	// Convert
	key := "certificates/test.txt"
	result := convertKey(key)
	if result != expect {
		t.Errorf("Convert failed for %q: got %q want %q", key, result, expect)
	}

	// Strip
	key = "/certificates/test.txt"
	result = convertKey(key)
	if result != expect {
		t.Errorf("Strip failed for %q: got %q want %q", key, result, expect)
	}
}

func TestGetSecretFileIntegration(t *testing.T) {
	secretKey := "/certificates/test.txt"

	vault := getApiStore()
	token := getToken()
	secret, err := getSecretFile(vault, token, secretKey)

	if err != nil {
		t.Errorf("Retrieving secret failed: %q", err)
	}
	if len(secret) == 0 {
		t.Errorf("Retrieved secret length wrong: got %q want %q", len(secret), ">0")
	}
}

func TestRetrieveSecret(t *testing.T) {
	// Variable
	{
		keyVariable := "smoketest"
		_, err := RetrieveSecret(keyVariable);
		if err != nil {
			t.Errorf("Failed getting secret from variable: %q", err)
		}
	}

	// File
	{
		keyFile := "/certificates/test.txt"
		_, err := RetrieveSecret(keyFile);
		if err != nil {
			t.Errorf("Failed getting secret from file: %q", err)
		}
	}

}
