package configs

var DefaultConfigs = `
service:
  host: localhost
  api_port: 8080
  device_port: 8081
  pid_file: /var/eywa/eywa.pid
security:
  dashboard:
    username: root
    password: waterISwide
    token_expiry: 24h
    aes:
      key: abcdefg123456789
      iv: 123456789abcdefg
  ssl:
    cert_file:
    key_file:
  api_key: dRiftingcLouds
connections:
  registry: memory
  nshards: 8
  init_shard_size: 1024
  http:
    timeouts:
      long_polling: 600s
  websocket:
    request_queue_size: 8
    timeouts:
      write: 4s
      read: 300s
      request: 4s
      response: 16s
    buffer_sizes:
      read: 1024
      write: 1024
indices:
  disable: false
  host: localhost
  port: 9200
  number_of_shards: 8
  number_of_replicas: 0
  ttl_enabled: false
  ttl: 0s
database:
  db_type: sqlite3
  db_file: /var/eywa/eywa.db
logging:
  eywa:
    filename: /var/eywa/eywa.log
    maxsize: 1024
    maxage: 7
    maxbackups: 5
    level: info
    buffer_size: 512
  indices:
    filename: /var/eywa/indices.log
    maxsize: 1024
    maxage: 7
    maxbackups: 5
    level: warn
    buffer_size: 512
  database:
    filename: /var/eywa/db.log
    maxsize: 1024
    maxage: 7
    maxbackups: 5
    level: warn
    buffer_size: 512
`
