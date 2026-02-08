# 建置階段
FROM golang:1.25-alpine AS builder

# Alpine 預設無 git，go mod download 需要
RUN apk add --no-cache git

WORKDIR /app

# 先複製 go.mod 和 go.sum（如果存在），下載依賴（利用快取）
COPY go.mod go.sum* ./
RUN go mod tidy && go mod download

# 複製其他源碼文件
COPY . .

# 建置應用程式
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /linebot .

# 執行階段（精簡映像）
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Taipei

WORKDIR /app
COPY --from=builder /linebot .

EXPOSE 8080
ENV PORT=8080

USER nobody
ENTRYPOINT ["/app/linebot"]
