curl -sS -X POST {{.server_url}}/node/api/request \
  -d "hostname=$(hostname)" \
  -d "name=$(hostname)" \
  -d "management_port=8000"
