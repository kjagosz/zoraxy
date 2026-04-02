package node

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNodeApprovalStateDefaultsToApproved(t *testing.T) {
	targetNode := &Node{}

	if targetNode.ApprovalState() != NodeApprovalStatusApproved {
		t.Fatalf("expected empty approval status to default to %q, got %q", NodeApprovalStatusApproved, targetNode.ApprovalState())
	}
	if !targetNode.IsApproved() {
		t.Fatalf("expected node with empty approval status to be approved")
	}
	if targetNode.IsPendingApproval() {
		t.Fatalf("expected node with empty approval status to not be pending")
	}
}

func TestGetNodeByTokenRejectsPendingNodes(t *testing.T) {
	manager := &Manager{
		Options: &Options{ConfigStore: t.TempDir()},
		Nodes: []*Node{
			{
				ID:             "pending-node",
				Token:          "pending-token",
				ApprovalStatus: NodeApprovalStatusPending,
			},
		},
	}

	_, err := manager.GetNodeByToken("pending-token")
	if !IsPendingApprovalError(err) {
		t.Fatalf("expected pending approval error, got %v", err)
	}
}

func TestApproveNodePersistsApprovedState(t *testing.T) {
	configDir := t.TempDir()
	targetNode := &Node{
		ID:             "approve-node",
		Token:          "approved-token",
		RegisteredAt:   time.Now().Add(-time.Minute),
		ApprovalStatus: NodeApprovalStatusPending,
	}
	manager := &Manager{
		Options: &Options{ConfigStore: configDir},
		Nodes:   []*Node{targetNode},
	}

	if err := manager.ApproveNode(targetNode); err != nil {
		t.Fatalf("ApproveNode returned error: %v", err)
	}
	if !targetNode.IsApproved() {
		t.Fatalf("expected node to be approved after ApproveNode")
	}
	if targetNode.ApprovedAt.IsZero() {
		t.Fatalf("expected ApproveNode to set ApprovedAt")
	}

	content, err := os.ReadFile(filepath.Join(configDir, targetNode.ID+".config"))
	if err != nil {
		t.Fatalf("failed to read persisted node config: %v", err)
	}

	savedNode := &Node{}
	if err := json.Unmarshal(content, savedNode); err != nil {
		t.Fatalf("failed to decode persisted node config: %v", err)
	}
	if savedNode.ApprovalState() != NodeApprovalStatusApproved {
		t.Fatalf("expected persisted node approval state %q, got %q", NodeApprovalStatusApproved, savedNode.ApprovalState())
	}
	if savedNode.ApprovedAt.IsZero() {
		t.Fatalf("expected persisted node to include ApprovedAt")
	}
}
