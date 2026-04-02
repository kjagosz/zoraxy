{{- with .node_name}}ZORAXY_NODE_NAME={{shellquote .}} {{end}}zoraxy -server {{shellquote .node_server}} -token {{shellquote .node_token}}
