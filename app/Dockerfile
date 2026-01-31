# Etap budowania aplikacji
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Skopiuj pliki zależności i katalog vendor
COPY go.mod go.sum ./
COPY vendor/ ./vendor
ENV GOFLAGS="-mod=vendor"

# Skopiuj źródła
COPY . .

# Buduj binarkę
RUN go build -o app .

# Etap końcowy – minimalny obraz
FROM alpine:latest

WORKDIR /app

# Skopiuj binarkę z buildera
COPY --from=builder /app/app .

# Skopiuj katalog logs (wymagany przez aplikację)
COPY logs/ ./logs/

# Domyślna komenda
CMD ["./app"]