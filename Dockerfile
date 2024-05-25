FROM golang:alpine

WORKDIR /app

RUN apk update && apk add git

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -o bin/updater cmd/updater/main.go

RUN cp -r ./bin/updater ./workdir

WORKDIR /app/workdir

RUN git config --global http.postBuffer 157286400

CMD ["./updater"]
