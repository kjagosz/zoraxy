package node

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleRequestNodeRegistrationIncludesInstructions(t *testing.T) {
	manager := &Manager{
		Options: &Options{
			Mode:        "primary",
			ConfigStore: t.TempDir(),
			BuildJoinInstructions: func(r *http.Request, targetNode *Node) ([]*JoinInstruction, error) {
				if targetNode == nil {
					t.Fatal("expected pending node to be passed to BuildJoinInstructions")
				}
				if strings.TrimSpace(targetNode.Token) == "" {
					t.Fatal("expected pending node token to be available to BuildJoinInstructions")
				}

				return []*JoinInstruction{
					{
						ID:      "dockerJoinEnvCommand",
						Title:   "Docker instructions (ENV)",
						Content: "docker run ...",
					},
				}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/node/api/request", strings.NewReader("hostname=worker-one&name=Worker+One&management_port=8000"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Host = "primary.example:8443"

	recorder := httptest.NewRecorder()
	manager.HandleRequestNodeRegistration(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d with body %q", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response struct {
		ID             string             `json:"id"`
		Name           string             `json:"name"`
		Host           string             `json:"host"`
		Token          string             `json:"token"`
		ApprovalStatus string             `json:"approval_status"`
		Approved       bool               `json:"approved"`
		Instructions   []*JoinInstruction `json:"instructions"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if strings.TrimSpace(response.ID) == "" {
		t.Fatal("expected response to include generated node id")
	}
	if response.Name != "Worker One" {
		t.Fatalf("expected response name %q, got %q", "Worker One", response.Name)
	}
	if response.Host != "worker-one" {
		t.Fatalf("expected response host %q, got %q", "worker-one", response.Host)
	}
	if strings.TrimSpace(response.Token) == "" {
		t.Fatal("expected response to include generated token")
	}
	if response.ApprovalStatus != NodeApprovalStatusPending {
		t.Fatalf("expected approval status %q, got %q", NodeApprovalStatusPending, response.ApprovalStatus)
	}
	if response.Approved {
		t.Fatal("expected dynamic registration response to stay unapproved")
	}
	if len(response.Instructions) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(response.Instructions))
	}
	if response.Instructions[0].ID != "dockerJoinEnvCommand" {
		t.Fatalf("expected instruction id %q, got %q", "dockerJoinEnvCommand", response.Instructions[0].ID)
	}
}
