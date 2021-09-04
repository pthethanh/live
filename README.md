# live
live reload for micro

*Note*: this project is just an experiment.

Use: `live watch.yml`

`watch.yml`
```
commands:
- dir: ./
  command: /usr/local/go/bin/go
  args:
  - run
  - main.go
watchers:
- enable: true
  targets:
  - ./
  - main.go
  - .env
  commands:
  - dir: ./
    command: /usr/bin/curl
    args:
      - http://localhost:8000/internal/shutdown
  - dir: ./
    command: sleep
    args:
      - 2s
  - dir: ./
    command: go
    args:
      - run
      - main.go
```
