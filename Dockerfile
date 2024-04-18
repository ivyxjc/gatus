# Build the go application into a binary
FROM golang:alpine as builder
RUN apk --update add ca-certificates
WORKDIR /app
COPY . ./
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gatus .

# Run Tests inside docker image if you don't have a configured go environment
#RUN apk update && apk add --virtual build-dependencies build-base gcc
#RUN go test ./... -mod vendor

# Run the binary on an empty container
FROM scratch as gatus
COPY --from=builder /app/gatus .
COPY --from=builder /app/config.yaml ./config/config.yaml
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt


FROM ubuntu as runner

RUN apt-get update \
    && apt-get install -y python3 \
                python3-pip \
                cron \
    && pip3 install --upgrade pip \
    && pip3 install --no-cache-dir \
            awscli \
    && rm -rf /var/lib/apt/lists/*

COPY --from=gatus /gatus .
COPY --from=gatus /config/config.yaml ./config/config.yaml
COPY --from=gatus /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY script.sh /script.sh
RUN chmod 744 /script.sh
RUN echo "*/1  *  *  *  * /script.sh >> /var/log/cron.log 2>&1" > /etc/cron.d/config-cron
RUN chmod 0644 /etc/cron.d/config-cron
RUN crontab /etc/cron.d/config-cron

RUN touch /var/log/cron.log
RUN touch /aws.env

ENV PORT=8080
EXPOSE ${PORT}
ENTRYPOINT bash /script.sh && cron && /gatus 2>&1