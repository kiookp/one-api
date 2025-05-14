# ========== 第一步：构建前端 ==========
FROM node:16 AS frontend

WORKDIR /web
COPY ./VERSION .
COPY ./web .

# 安装依赖（缓存优化）
WORKDIR /web/default
COPY ./web/default/package*.json ./
RUN npm install

# 构建 default 主题
COPY ./web/default ./
RUN DISABLE_ESLINT_PLUGIN=true REACT_APP_VERSION=$(cat /web/VERSION) npm run build

# 拷贝构建结果到统一路径
RUN mkdir -p /web/build && cp -r build/* /web/build/

# ========== 第二步：构建后端 ==========
FROM golang:alpine AS backend

RUN apk add --no-cache \
    gcc \
    musl-dev \
    sqlite-dev \
    build-base

ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /web/build ./web/build

RUN go build -trimpath -ldflags "-s -w -X 'github.com/songquanpeng/one-api/common.Version=$(cat VERSION)' -linkmode external -extldflags '-static'" -o one-api

# ========== 第三步：最小运行镜像 ==========
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

COPY --from=backend /build/one-api /

EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/one-api"]