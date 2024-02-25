FROM golang:1.21.6

WORKDIR /go/src/app

COPY . .

RUN go build -o main main.go

CMD ["./main"]