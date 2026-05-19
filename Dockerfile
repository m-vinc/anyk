FROM golang:1.26-bookworm

WORKDIR /build

COPY go.mod go.sum /build

RUN go mod download
COPY . /build

RUN mkdir -p dist && go build -o /build/dist/anyk

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y curl gnupg 
RUN curl -s https://deb.frrouting.org/frr/keys.gpg | tee /usr/share/keyrings/frrouting.gpg > /dev/null

# MUST MATCH FRR used by the host to avoid weird bugs 
ENV FRRVER="frr-9.1"
RUN echo deb '[signed-by=/usr/share/keyrings/frrouting.gpg]' https://deb.frrouting.org/frr bookworm $FRRVER | tee -a /etc/apt/sources.list.d/frr.list
RUN apt update && apt install -y frr frr-pythontools

COPY --from=0 /build/dist/anyk /usr/bin/anyk

ENTRYPOINT ["/usr/bin/anyk"]
