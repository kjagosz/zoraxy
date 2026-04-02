docker volume create zoraxy-config || true
docker stop -s 9 || true
docker rm -f zoraxy || true
docker run -d --network=host --name=zoraxy --volume zoraxy-config:/opt/zoraxy/config --restart=always -e ZORAXY_NODE_SERVER={{shellquote .node_server}} -e ZORAXY_NODE_TOKEN={{shellquote .node_token}} {{- with .node_name}}  -e ZORAXY_NODE_NAME={{shellquote .}}{{end}}  {{.docker_image}}
