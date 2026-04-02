package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"imuslab.com/zoraxy/mod/node"
	"imuslab.com/zoraxy/mod/utils"
)

const (
	nodeJoinManifestResourcePath = "web/node-join/index.json"
	nodeJoinDynamicRequestName   = "dynamic_request"
	nodeJoinOptionalIPCommandID  = "nodeIPJoinCommand"
)

type nodeJoinTemplateDefinition struct {
	ID    string `json:"id,omitempty"`
	Title string `json:"title,omitempty"`
	Path  string `json:"path"`
}

type nodeJoinTemplateManifest struct {
	DynamicRequest *nodeJoinTemplateDefinition   `json:"dynamic_request"`
	Instructions   []*nodeJoinTemplateDefinition `json:"instructions"`
}

type nodeJoinRenderedTemplatesResponse struct {
	Instructions []*node.JoinInstruction `json:"instructions,omitempty"`
}

func normalizeNodeJoinResourcePath(resourcePath string) (string, error) {
	resourcePath = strings.TrimSpace(resourcePath)
	if resourcePath == "" {
		return "", fmt.Errorf("empty node join template path")
	}

	resourcePath = strings.TrimPrefix(resourcePath, "/")
	resourcePath = filepath.ToSlash(resourcePath)
	if strings.HasPrefix(resourcePath, "web/") {
		return resourcePath, nil
	}
	if !strings.HasPrefix(resourcePath, "node-join/") {
		return "", fmt.Errorf("invalid node join template path")
	}

	return filepath.ToSlash(filepath.Join("web", resourcePath)), nil
}

func readNodeJoinResource(resourcePath string) ([]byte, error) {
	normalizedPath, err := normalizeNodeJoinResourcePath(resourcePath)
	if err != nil {
		return nil, err
	}

	if *development_build {
		return os.ReadFile(normalizedPath)
	}

	return webres.ReadFile(normalizedPath)
}

func loadNodeJoinTemplateManifest() (*nodeJoinTemplateManifest, error) {
	rawManifest, err := readNodeJoinResource(nodeJoinManifestResourcePath)
	if err != nil {
		return nil, err
	}

	manifest := &nodeJoinTemplateManifest{}
	if err := json.Unmarshal(rawManifest, manifest); err != nil {
		return nil, err
	}

	if manifest.DynamicRequest == nil {
		manifest.DynamicRequest = &nodeJoinTemplateDefinition{
			ID:   nodeJoinDynamicRequestName,
			Path: "/node-join/dynamic-request.sh",
		}
	}
	if strings.TrimSpace(manifest.DynamicRequest.ID) == "" {
		manifest.DynamicRequest.ID = nodeJoinDynamicRequestName
	}

	return manifest, nil
}

func getNodeJoinServerURL(r *http.Request) string {
	scheme := "http"
	if r != nil && r.TLS != nil {
		scheme = "https"
	}
	if r != nil {
		if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
			scheme = strings.TrimSpace(strings.Split(forwardedProto, ",")[0])
		}
	}

	host := ""
	if r != nil {
		host = strings.TrimSpace(r.Host)
	}

	return strings.TrimRight(scheme+"://"+host, "/")
}

func shellEscapeNodeJoinValue(value string) string {
	return "'" + strings.ReplaceAll(strings.TrimSpace(value), "'", "'\"'\"'") + "'"
}

func yamlQuoteNodeJoinValue(value string) string {
	quoted, _ := json.Marshal(strings.TrimSpace(value))
	return string(quoted)
}

func getNodeJoinDisplayName(targetNode *node.Node) string {
	if targetNode == nil {
		return ""
	}

	if strings.TrimSpace(targetNode.Name) != "" {
		return strings.TrimSpace(targetNode.Name)
	}
	if strings.TrimSpace(targetNode.Host) != "" {
		return strings.TrimSpace(targetNode.Host)
	}

	return strings.TrimSpace(targetNode.ID)
}

func buildNodeJoinTemplateData(r *http.Request, targetNode *node.Node) map[string]any {
	serverURL := getNodeJoinServerURL(r)
	dockerImage, _ := getConfiguredNodeDockerImage()
	if dockerImage == "" {
		dockerImage = "zoraxydocker/zoraxy:latest"
	}

	nodeServer := ""
	nodeToken := ""
	nodeName := ""

	if targetNode != nil {
		nodeServer = serverURL
		nodeToken = strings.TrimSpace(targetNode.Token)
		nodeName = getNodeJoinDisplayName(targetNode)
	}

	return map[string]any{
		"server_url":      serverURL,
		"docker_image":    dockerImage,
		"node_server":     nodeServer,
		"node_token":      nodeToken,
		"node_name":       nodeName,
		"node_ip_example": "192.0.2.10",
	}
}

func getNodeJoinTemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"shellquote": shellEscapeNodeJoinValue,
		"yamlquote":  yamlQuoteNodeJoinValue,
	}
}

func renderNodeJoinTemplate(definition *nodeJoinTemplateDefinition, data map[string]any) (string, error) {
	if definition == nil {
		return "", fmt.Errorf("missing node join template definition")
	}

	templateContent, err := readNodeJoinResource(definition.Path)
	if err != nil {
		return "", err
	}

	renderer, err := template.New(definition.ID).Funcs(getNodeJoinTemplateFuncMap()).Option("missingkey=zero").Parse(string(templateContent))
	if err != nil {
		return "", err
	}

	buffer := &bytes.Buffer{}
	if err := renderer.Execute(buffer, data); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

func resolveNodeJoinTemplateDefinition(manifest *nodeJoinTemplateManifest, name string, index int) (*nodeJoinTemplateDefinition, bool, error) {
	if manifest == nil {
		return nil, false, fmt.Errorf("missing node join manifest")
	}

	if name != "" {
		if name == nodeJoinDynamicRequestName {
			return manifest.DynamicRequest, true, nil
		}
		for _, definition := range manifest.Instructions {
			if definition != nil && strings.TrimSpace(definition.ID) == name {
				return definition, false, nil
			}
		}
		return nil, false, fmt.Errorf("node join template not found")
	}

	if index < 0 || index >= len(manifest.Instructions) {
		return nil, false, fmt.Errorf("node join template index out of range")
	}

	return manifest.Instructions[index], false, nil
}

func renderNodeJoinTemplateResponse(r *http.Request, targetNode *node.Node, definition *nodeJoinTemplateDefinition) (*node.JoinInstruction, error) {
	content, err := renderNodeJoinTemplate(definition, buildNodeJoinTemplateData(r, targetNode))
	if err != nil {
		return nil, err
	}

	return &node.JoinInstruction{
		ID:      definition.ID,
		Title:   definition.Title,
		Content: content,
	}, nil
}

func renderNodeJoinInstructionDefinitions(r *http.Request, targetNode *node.Node, definitions []*nodeJoinTemplateDefinition, excludedIDs map[string]struct{}) ([]*node.JoinInstruction, error) {
	renderedInstructions := make([]*node.JoinInstruction, 0, len(definitions))
	for _, definition := range definitions {
		if definition == nil {
			continue
		}
		if excludedIDs != nil {
			if _, excluded := excludedIDs[strings.TrimSpace(definition.ID)]; excluded {
				continue
			}
		}

		renderedTemplate, err := renderNodeJoinTemplateResponse(r, targetNode, definition)
		if err != nil {
			return nil, err
		}
		renderedInstructions = append(renderedInstructions, renderedTemplate)
	}

	return renderedInstructions, nil
}

func renderNodeJoinInstructions(r *http.Request, targetNode *node.Node, excludedIDs map[string]struct{}) ([]*node.JoinInstruction, error) {
	manifest, err := loadNodeJoinTemplateManifest()
	if err != nil {
		return nil, err
	}

	return renderNodeJoinInstructionDefinitions(r, targetNode, manifest.Instructions, excludedIDs)
}

func HandleNodeJoinTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if *mode != "primary" {
		http.Error(w, "Node join templates are only available on the primary node", http.StatusServiceUnavailable)
		return
	}

	manifest, err := loadNodeJoinTemplateManifest()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	name := strings.TrimSpace(r.URL.Query().Get("name"))
	index := -1
	if indexValue := strings.TrimSpace(r.URL.Query().Get("index")); indexValue != "" {
		parsedIndex, parseErr := strconv.Atoi(indexValue)
		if parseErr != nil {
			http.Error(w, "invalid index given", http.StatusBadRequest)
			return
		}
		index = parsedIndex
	}

	if name != "" || index >= 0 {
		definition, isDynamicRequest, err := resolveNodeJoinTemplateDefinition(manifest, name, index)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		var targetNode *node.Node
		if !isDynamicRequest {
			nodeID, err := utils.GetPara(r, "id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			targetNode, err = nodeManager.GetNodeByID(nodeID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		renderedTemplate, err := renderNodeJoinTemplateResponse(r, targetNode, definition)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response, err := json.Marshal(renderedTemplate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		utils.SendJSONResponse(w, string(response))
		return
	}

	nodeID, err := utils.GetPara(r, "id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	targetNode, err := nodeManager.GetNodeByID(nodeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	renderedInstructions, err := renderNodeJoinInstructionDefinitions(r, targetNode, manifest.Instructions, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response, err := json.Marshal(&nodeJoinRenderedTemplatesResponse{
		Instructions: renderedInstructions,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	utils.SendJSONResponse(w, string(response))
}
