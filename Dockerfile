# ========== 第一步：构建前端 ==========
FROM node:16 AS frontend

# 拷贝 VERSION 和前端代码
WORKDIR /web
COPY ./VERSION .
COPY ./web/default ./default

# 安装依赖（先单独拷贝以缓存）
WORKDIR /web/default
COPY ./web/default/package*.json ./
RUN npm install

# 构建 default 主题
COPY ./web/default ./
RUN DISABLE_ESLINT_PLUGIN=true REACT_APP_VERSION=$(cat ../VERSION) npm run build

# ✅ 拷贝构建结果到指定目录
RUN mkdir -p /web/build/default && cp -r build/* /web/build/default/

# ========== 第二步：构建后端 ==========
FROM golang:alpine AS backend

RUN apk add --no-cache gcc musl-dev sqlite-dev build-base

ENV GO111MODULE=on \
    CGO_ENABLED=1 \
    GOOS=linux

WORKDIR /build

# 优先 COPY 依赖文件以缓存构建
COPY go.mod go.sum ./
RUN go mod download

# 拷贝剩余后端代码（包含 VERSION）
COPY . .

# 拷贝前端构建产物
COPY --from=frontend /web/build ./web/build

# ✅ 编译后端并注入版本号
RUN go build -trimpath -ldflags "-s -w -X 'github.com/songquanpeng/one-api/common.Version=$(cat VERSION)' -linkmode external -extldflags '-static'" -o one-api

# ========== 第三步：最小运行镜像 ==========
FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata

COPY --from=backend /build/one-api /

EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/one-api"]