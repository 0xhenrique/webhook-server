package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type GitHubPayload struct {
	Ref        string `json:"ref"`
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
}

// Map repo names to their deploy scripts
var deployScripts = map[string]string{
	"0xhenrique-blog":    "/srv/0xhenrique-blog/deploy.sh",
	"agora":   "/srv/agora/deploy.sh",
	"agora-backend":  "/srv/agora-backend/deploy.sh",
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload GitHubPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("Failed to decode payload: %v", err)
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Extract branch name from ref (refs/heads/master -> master)
	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
	repoName := payload.Repository.Name

	log.Printf("Received webhook: repo=%s, branch=%s", repoName, branch)

	// Only deploy on master branch
	if branch != "master" {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Ignoring branch: %s\n", branch)
		return
	}

	// Find deploy script for this repo
	deployScript, exists := deployScripts[repoName]
	if !exists {
		log.Printf("No deploy script configured for repo: %s", repoName)
		http.Error(w, "Unknown repository", http.StatusNotFound)
		return
	}

	// Check if deploy script exists
	if _, err := os.Stat(deployScript); os.IsNotExist(err) {
		log.Printf("Deploy script not found: %s", deployScript)
		http.Error(w, "Deploy script not found", http.StatusInternalServerError)
		return
	}

	// Execute deploy script
	log.Printf("Executing deploy script: %s", deployScript)
	cmd := exec.Command("/run/current-system/profile/bin/bash", deployScript)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("Deploy failed for %s: %v\nOutput: %s", repoName, err, output)
		http.Error(w, fmt.Sprintf("Deploy failed: %s", output), http.StatusInternalServerError)
		return
	}

	log.Printf("Deploy successful for %s\nOutput: %s", repoName, output)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Deploy successful for %s\n", repoName)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9000"
	}

	http.HandleFunc("/webhook", handleWebhook)

	log.Printf("Webhook server listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
