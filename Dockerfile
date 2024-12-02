FROM golang:1.18 as compiler

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct
RUN apt-get update && apt-get install -y tzdata

RUN apt-get update && apt-get install -y tzdata

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o transfer .

RUN mkdir publish && cp transfer publish && \
    cp app.yml publish && cp -r web/statics publish

# 第二阶段
FROM alpine

WORKDIR /app

COPY --from=compiler /app/publish .
# 安装时区数据
RUN apk add --no-cache tzdata

# 设置时区为 Asia/Shanghai
ENV TZ=Asia/Shanghai

# 注意修改端口
EXPOSE 9595

ENTRYPOINT ["./transfer"]