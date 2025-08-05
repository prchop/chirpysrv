OUT = ./out

build:
	@go build -o ${OUT} && ${OUT}
