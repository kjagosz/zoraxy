package node

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	NodeApprovalStatusApproved = "approved"
	NodeApprovalStatusPending  = "pending"
)

var ErrNodeApprovalPending = errors.New("node registration pending approval")

func normalizeNodeApprovalStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case NodeApprovalStatusPending:
		return NodeApprovalStatusPending
	default:
		return NodeApprovalStatusApproved
	}
}

func (n *Node) ApprovalState() string {
	if n == nil {
		return NodeApprovalStatusApproved
	}

	return normalizeNodeApprovalStatus(n.ApprovalStatus)
}

func (n *Node) IsApproved() bool {
	return n.ApprovalState() == NodeApprovalStatusApproved
}

func (n *Node) IsPendingApproval() bool {
	return n.ApprovalState() == NodeApprovalStatusPending
}

func (m *Manager) GenerateDynamicJoinRequest(name string, hostname string, managementPort string, requestIP string) (*Node, error) {
	node := &Node{
		ID:             uuid.New().String(),
		Name:           strings.TrimSpace(name),
		Host:           strings.TrimSpace(hostname),
		RequestIP:      strings.TrimSpace(requestIP),
		ManagementPort: strings.TrimSpace(managementPort),
		Enabled:        true,
		Token:          generateNodeToken(),
		RegisteredAt:   time.Now(),
		ApprovalStatus: NodeApprovalStatusPending,
	}
	if err := m.RegisterNode(node); err != nil {
		return nil, err
	}
	return node, nil
}

func (m *Manager) ApproveNode(targetNode *Node) error {
	if targetNode == nil {
		return fmt.Errorf("node cannot be nil")
	}

	targetNode.ApprovalStatus = NodeApprovalStatusApproved
	if targetNode.ApprovedAt.IsZero() {
		targetNode.ApprovedAt = time.Now()
	}
	m.SaveConfigToDatabase()
	return nil
}
