.PHONY: build/schemas dev.start docker.login docker.build docker.tag docker.push deployment.apply deployment.list terraform.shell

ECR_REPO_URL=XXXXXXX.dkr.ecr.us-west-1.amazonaws.com
KHULNASOFT-LAB=$${PWD%/khulnasoft-lab/**}/khulnasoft-lab
ECR_PACKAGE_PREFIX=t-fastnode-com
TAG=$(shell git rev-parse --short HEAD)
CLUSTER_NAME=t-fastnode-com
GO_IMAGE=golang:1.15.3-buster

DOCKER_TERRAFORM_ARGS=-w /opt/terraform -v $(PWD)/deployment/terraform:/opt/terraform -v $(HOME)/.gcloud/:/root/.gcloud -v $(HOME)/.aws/:/root/.aws -v $(HOME)/.config/gcloud/:/root/.config/gcloud

dev.start: build/fastnode.khulnasoft.com build/schemas
	docker-compose up -d

dev.build: build/fastnode.khulnasoft.com build/schemas
	docker-compose restart webserver

build/fastnode.khulnasoft.com: build main.go
	docker run --rm -e "GOPRIVATE=github.com/khulnasoft-lab/*" \
		-v $(KHULNASOFT-LAB):/go/src/github.com/khulnasoft-lab/fastnode \
		-v $(PWD)/build:/build \
		-w /go/src/github.com/khulnasoft-lab/fastnode/services/t.khulnasoft.com/ \
		$(GO_IMAGE) go build -o /build/t.khulnasoft.com .

docker.login:
	aws ecr get-login-password --region us-west-1 | docker login --username AWS --password-stdin $(ECR_REPO_URL)

docker.build: build/schemas build/fastnode.khulnasoft.com
	docker build -t $(ECR_PACKAGE_PREFIX)-webserver:$(TAG) .
	docker build -t $(ECR_PACKAGE_PREFIX)-fluentd:$(TAG) ./fluentd

docker.tag:
	docker tag $(ECR_PACKAGE_PREFIX)-webserver:$(TAG) $(ECR_REPO_URL)/$(ECR_PACKAGE_PREFIX)-webserver:$(TAG)
	docker tag $(ECR_PACKAGE_PREFIX)-fluentd:$(TAG) $(ECR_REPO_URL)/$(ECR_PACKAGE_PREFIX)-fluentd:$(TAG)

docker.push:
	docker push $(ECR_REPO_URL)/$(ECR_PACKAGE_PREFIX)-webserver:$(TAG)
	docker push $(ECR_REPO_URL)/$(ECR_PACKAGE_PREFIX)-fluentd:$(TAG)

docker.all: docker.login docker.build docker.tag docker.push

build/schemas:
	mkdir -p build/schemas
	pipenv install -e $(KHULNASOFT-LAB)/fastnode-python/metrics
	pipenv run fastnode-metrics-schemas --out ./build/schemas

deployment.apply:
	docker run $(DOCKER_TERRAFORM_ARGS) hashicorp/terraform:latest workspace select production
	docker run $(DOCKER_TERRAFORM_ARGS) hashicorp/terraform:latest apply -var tag=$(TAG) -auto-approve

deployment.list:
	@aws ecs list-tasks --cluster $(CLUSTER_NAME) | jq -r '.taskArns[]' | xargs aws ecs describe-tasks --cluster $(CLUSTER_NAME) --tasks | jq -r '.tasks[] | {"group": .group, "container": .containers[]} | (.group + "    " + .container.name + "    " + (.container.image | capture(".*:(?<tag>[0-9a-f]+)$$") | .tag) + "    " + .container.lastStatus)'

terraform.shell:
	docker run $(DOCKER_TERRAFORM_ARGS) --entrypoint sh -it hashicorp/terraform:latest

test-client:
	go build -o test-client ./test

build:
	mkdir -p build

clean:
	rm -rf build/*