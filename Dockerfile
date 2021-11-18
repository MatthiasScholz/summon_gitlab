###############################################
# Shared Build Time Arguments
###############################################
ARG app_name=summon-gitlab
# NOTE golang has a folder naming convention:
ARG build_dir=/src

###############################################
# Builder
###############################################
FROM docker.io/library/golang:1.16.6-alpine as builder
ARG app_name
ARG build_dir

WORKDIR ${build_dir}

# Get Summon Core
ARG SUMMON_VERSION=v0.9.0
ADD https://github.com/cyberark/summon/releases/download/${SUMMON_VERSION}/summon-linux-amd64.tar.gz ${build_dir}/
RUN tar -xf ${build_dir}/summon-linux-amd64.tar.gz

# Copy only relevant files
COPY *.go ./
COPY go.mod .

# Test
# NOTE The Alpine base images does not contain gcc.
#ARG CGO_ENABLED=0
#RUN go test -v -coverprofile=app.coverage
#RUN go tool cover -func=app.coverage

# Build Summon Provider
RUN go build -o ${app_name}

###############################################
# Final
###############################################
FROM docker.io/library/alpine:3.14
# NOTE: Only for troubleshooting connection issues
ARG app_name
ARG build_dir
ENV app_name=${app_name}

ARG tag=-
ARG build_time=-

LABEL Tag=${tag}
LABEL BuildTime=${build_time}

ENV TAG=${tag}
ENV BUILD_TIME=${build_time}

WORKDIR /opt/

COPY --from=builder ${build_dir}/${app_name} .
COPY --from=builder ${build_dir}/summon .

# TEST
RUN ls -la /opt/summon

ENV SUMMON_PROVIDER_PATH="/opt"
ENV GITLAB_TOKEN="UNSET"
ENTRYPOINT /opt/summon --provider /opt/${app_name} -f /mnt/secrets.yml printenv
