default: clean
	@export GOPATH=$$GOPATH:$$(pwd) && go install rmg
edit:
	@export GOPATH=$$GOPATH:$$(pwd) && atom .
edit2:
	@export GOPATH=$$GOPATH:$$(pwd) && code .
run: default
	@bin/rmg
	@echo ""

run2: default
	@bin/rmg --env=staging --elb-type=api --path=notifications
	@echo ""

clean:
	@rm -rf bin

pua: test
	git checkout master && git merge dev && git checkout dev && git push origin --all

test: default
	@export GOPATH=$$GOPATH:$$(pwd) && go test ./...
test_ver:
	@export GOPATH=$$GOPATH:$$(pwd) && go test -v ./...
setup:
	go get -u github.com/aws/aws-sdk-go/...
