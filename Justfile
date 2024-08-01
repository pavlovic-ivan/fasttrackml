# Project-specific variables
set shell := ['sh', '-c']

APP := if `go env GOOS` == 'windows' { 'fml.exe' } else { 'fml' }
VERSION := if `git describe --tags --dirty --match='v*' 2> /dev/null | sed 's/^v//'` == '' { '0.0.0-g' + `git describe --always --dirty 2> /dev/null` } else { '' }
GO_LDFLAGS := '-s -w -X github.com/G-Research/fasttrackml/pkg/version.Version=' + VERSION
GO_BUILDTAGS := `cat .go-build-tags 2> /dev/null`

ARCHIVE_EXT := if `go env GOOS` == 'windows' { 'zip' } else { 'tar.gz' }
ARCHIVE_CMD := if `go env GOOS` == 'windows' {
    'zip -r'
} else if `which gtar >/dev/null 2>/dev/null; echo $?` == '0' {
    'gtar -czf'
} else {
    'tar -czf'
}

ARCHIVE_NAME := 'dist/fasttrackml_' + `go env GOOS | sed s/darwin/macos/` + '_' + `go env GOARCH | sed s/amd64/x86_64/` + '.' + ARCHIVE_EXT
ARCHIVE_FILES := APP + ' LICENSE README.md'

COMPOSE_FILE := 'tests/integration/docker-compose.yml'
COMPOSE_PROJECT_NAME := APP + '-integration-tests'

AIM_BUILD_LOCATION := '$HOME/fasttrackml-ui-aim'
MLFLOW_BUILD_LOCATION := '$HOME/fasttrackml-ui-mlflow'

# Default target (help)
@default:
    echo 'Please use `just <target>` where <target> is one of:'
    echo
    just --list

# Tools targets
install-tools:
    echo '>>> Installing tools.'
    go install github.com/vektra/mockery/v2@v2.34.0
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.54.2
    go install golang.org/x/tools/cmd/goimports@v0.13.0
    go install mvdan.cc/gofumpt@v0.5.0

# Linter targets
lint: go-lint python-lint

# Go targets
go-get:
    echo '>>> Getting go modules.'
    go mod download

go-build:
    echo '>>> Building go binary.'
    CGO_ENABLED=1 go build -ldflags={{GO_LDFLAGS}} -tags={{GO_BUILDTAGS}} -o {{APP}} ./main.go

go-format:
    echo '>>> Formatting go code.'
    gofumpt -w .
    goimports -w -local github.com/G-Research/fasttrackml `find . -type f -name '*.go' -not -name 'mock_*.go'`

go-lint:
    echo '>>> Running go linters.'
    golangci-lint run -v --build-tags {{GO_BUILDTAGS}}

go-dist: go-build
    echo '>>> Archiving go binary.'
    dir=`dirname {{ARCHIVE_NAME}}`; if [ ! -d $$dir ]; then mkdir -p $$dir; fi
    if [ -f {{ARCHIVE_NAME}} ]; then rm -f {{ARCHIVE_NAME}}; fi
    {{ARCHIVE_CMD}} {{ARCHIVE_NAME}} {{ARCHIVE_FILES}}

# Python targets
python-env:
    echo '>>> Creating python virtual environment.'
    pipenv sync

python-dist: go-build python-env
    echo '>>> Building Python Wheels.'
    VERSION={{VERSION}} pipenv run python3 -m pip wheel ./python --wheel-dir=wheelhouse --no-deps

python-format: python-env
    echo '>>> Formatting python code.'
    pipenv run black --line-length 120 .
    pipenv run isort --profile black .

python-lint: python-env
    echo '>>> Checking python code formatting.'
    pipenv run black --check --line-length 120 .
    pipenv run isort --check-only --profile black .

# Tests targets
test: test-go-unit container-test test-python-integration

test-go-unit:
    echo '>>> Running unit tests.'
    go test -tags={{GO_BUILDTAGS}} ./pkg/...

test-go-integration:
    echo '>>> Running integration tests.'
    go test -tags={{GO_BUILDTAGS}} ./tests/integration/golang/...

test-python-integration:
    echo '>>> Running all python integration tests.'
    go run tests/integration/python/main.go

test-python-integration-mlflow:
    echo '>>> Running MLFlow python integration tests.'
    go run tests/integration/python/main.go -targets mlflow

test-python-integration-aim:
    echo '>>> Running Aim python integration tests.'
    go run tests/integration/python/main.go -targets aim

# Container test targets
container-test:
    echo '>>> Running integration tests in container.'
    COMPOSE_FILE={{COMPOSE_FILE}} COMPOSE_PROJECT_NAME={{COMPOSE_PROJECT_NAME}} \
    docker compose run -e FML_SLOW_TESTS_ENABLED integration-tests

container-clean:
    echo '>>> Cleaning containers.'
    COMPOSE_FILE={{COMPOSE_FILE}} COMPOSE_PROJECT_NAME={{COMPOSE_PROJECT_NAME}} \
    docker compose down -v --remove-orphans

# Mockery targets
mocks-clean:
    echo '>>> Cleaning mocks.'
    find ./pkg -name 'mock_*.go' -type f -delete

mocks-generate: mocks-clean
    echo '>>> Generating mocks.'
    mockery

# Docker targets (Only available on Linux)
#if `go env GOOS` == 'linux' {
#    DOCKER_OUTPUT?=type=docker
#
#}
DOCKER_OUTPUT := if `go env GOOS` == 'linux' { env_var_or_default('DOCKER_OUTPUT', 'type=docker') } else { '' }

# setting DOCKER_METADATA to empty string because function env_var(env) will fail instead of returning a value
# so we can test the value and set DOCKER_<env> vars
DOCKER_METADATA := env_var_or_default('DOCKER_METADATA', '')

DOCKER_LABELS := if DOCKER_METADATA == '' {
    `echo $DOCKER_METADATA | jq -r '.labels | to_entries | map("--label \(.key)=\(.value)") | join(" ")'`
} else { '' }

DOCKER_TAGS := if DOCKER_METADATA == '' {
    `echo $$DOCKER_METADATA | jq -r '.tags | map("--tag \(.)") | join(" ")'`
} else {
    env_var_or_default('DOCKER_TAGS','fasttrackml:{{VERSION}}) fasttrackml:latest')
    DOCKER_TAGS:=$(addprefix --tag ,$(DOCKER_TAGS))
}

debug:
    echo '>>> Debugging.'
    echo 'DOCKER_OUTPUT: {{DOCKER_OUTPUT}}'
    echo 'DOCKER_METADATA: {{DOCKER_METADATA}}'
    echo 'DOCKER_LABELS: {{DOCKER_LABELS}}'

[linux]
docker-dist: go-build
    echo '>>> Building Docker image.'
    docker buildx build --provenance false --sbom false --platform linux/`go env GOARCH` --output {{DOCKER_OUTPUT}} {{DOCKER_TAGS}} {{DOCKER_LABELS}} .

# Build targets
clean:
    echo '>>> Cleaning build artifacts.'
    rm -rf {{APP}} dist wheelhouse

build: go-build

dist: go-dist python-dist
    if `go env GOOS` == 'linux' { just docker-dist }

format: go-format python-format

run: build
    echo '>>> Running the FasttrackML server.'
    ./{{APP}} server

migrations-create:
    echo '>>> Running FastTrackML migrations create.'
    go run main.go migrations create

migrations-rebuild:
    echo '>>> Running FastTrackML migrations rebuild.'
    go run main.go migrations rebuild

ui-aim-sync:
    echo '>>> Syncing the Aim UI.'
    rsync -rvu --exclude node_modules --exclude .git ui/fasttrackml-ui-aim/ {{AIM_BUILD_LOCATION}}

ui-aim-start: ui-aim-sync
    echo '>>> Starting the Aim UI.'
    cd {{AIM_BUILD_LOCATION}}/src && npm ci --legacy-peer-deps && npm start

ui-mlflow-sync:
    echo '>>> Syncing the MLflow UI.'
    rsync -rvu --exclude node_modules --exclude .git ui/fasttrackml-ui-mlflow/ {{MLFLOW_BUILD_LOCATION}}

ui-mlflow-start: ui-mlflow-sync
    echo '>>> Starting the MLflow UI.'
    cd {{MLFLOW_BUILD_LOCATION}}/src && yarn install --immutable && yarn start
