# Build the go application into a binary
FROM golang:alpine AS builder
RUN apk --update add ca-certificates
WORKDIR /app
COPY . ./
RUN go mod tidy -diff
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gatus .

# Run Tests inside docker image if you don't have a configured go environment
#RUN apk update && apk add --virtual build-dependencies build-base gcc
#RUN go test ./... -mod vendor

# Run the binary on an empty container
FROM scratch AS gatus
COPY --from=builder /app/gatus .
COPY --from=builder /app/config.yaml ./config/config.yaml
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

FROM ubuntu:24.04 AS runner

RUN set -eux \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
                ca-certificates \
                cron \
                curl \
                unzip \
    && ARCH="$(uname -m)" \
    && case "$ARCH" in \
         x86_64)  AWSARCH=x86_64 ;; \
         aarch64) AWSARCH=aarch64 ;; \
         *) echo "unsupported arch: $ARCH"; exit 1 ;; \
       esac \
    && curl -fsSL "https://awscli.amazonaws.com/awscli-exe-linux-${AWSARCH}.zip" -o /tmp/awscliv2.zip \
    && unzip -q /tmp/awscliv2.zip -d /tmp \
    && /tmp/aws/install \
    && rm -rf /tmp/aws /tmp/awscliv2.zip \
    && apt-get purge -y --auto-remove curl unzip \
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

ENV GATUS_CONFIG_PATH=""
ENV GATUS_LOG_LEVEL="INFO"
ENV PORT="8080"

ENV PORT=8080
EXPOSE ${PORT}
ENTRYPOINT bash /script.sh && cron && /gatus 2>&1