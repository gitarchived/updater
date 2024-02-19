FROM golang:alpine

WORKDIR /app

RUN apk update && apk add git

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o ./app 

CMD ["./app", "--events"]
