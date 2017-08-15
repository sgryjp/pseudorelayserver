build:
	go build .

test:
	go test .

vtest:
	go test -v .

racetest:
	go test -race .

install:
	go install .
