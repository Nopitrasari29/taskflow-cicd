# 🚀 TaskFlow CI/CD Pipeline — Kelompok 4

> **Mata Kuliah**: Operasional Pengembang (DevOps) 
> **Institut Teknologi Sepuluh Nopember (ITS)**  
> **Tool CI/CD**: GitHub Actions | **Fokus**: Multi-job dengan dependency graph

---

## 👥 Anggota Kelompok

| Nama | NRP | Area | Scope |
|------|-----|------|-------|
| Aswalia Novitriasari | 5027231012 | **Area 1** | S1 Bug Fix & Testing + S2 CI Pipeline |
| Riskiyatul Nur Oktarani | 5027231013 | **Area 4** | S6 Security Audit + Koordinasi Laporan |
| Rafika Az Zahra Kusumastuti | 5027231050 | **Area 2** | S3 Docker Build + Push ke GHCR |
| Nisrina Atiqah Dwiputri Ridzki | 5027231075 | **Area 3** | S4 Smoke Test + Notifikasi + S5 Rollback |

---

## 📋 Tentang Project

Project ini adalah implementasi CI/CD pipeline otomatis untuk **TaskFlow API** — REST API manajemen task berbasis Go — sebagai bagian dari PBL (Problem-Based Learning) mata kuliah Operasional Pengembang.

Pipeline dibangun menggunakan **GitHub Actions** dengan struktur multi-job dan dependency graph eksplisit, mencakup:
- ✅ Bug fix + automated testing
- ✅ CI pipeline dengan lint, unit test, integration test, dan coverage gate
- ✅ Docker multi-stage build + push ke GHCR
- ✅ Smoke test otomatis + notifikasi Telegram & Slack
- ✅ Rollback strategy dengan tag `stable`
- ✅ Security scanning (SAST + Secret Scanning)

---

## 🏗️ Struktur Pipeline

```
push/PR ke main atau develop
           │
           ▼
        [lint]
     go vet (analisis statis)
           │
     ┌─────┴─────┐
     ▼           ▼
[unit-test]  [integration-test]    [security-scan]
 -race         PostgreSQL 16            gosec
 -count=1      service container      + gitleaks
     │               │
     └──────┬─────────┘
            ▼
     [coverage-gate]
      cek coverage ≥ 75%
      upload artifact cov.out
            │
            ▼
     [build-binary]
      go build → bin/taskflow-api
            │
            ▼
    [docker-build-push]
     multi-stage build
     tag sha-xxxxxxx → GHCR
            │
            ▼
      [smoke-test]
    curl /health + /api/v1/stats
    update tag :stable jika PASS
            │
            ▼
        [notify]
    ✅ Telegram + Slack (sukses/gagal)
```
<img width="559" height="517" alt="image" src="https://github.com/user-attachments/assets/fb139ef9-9f2b-4d39-a26f-7f6a7969bced" />

---

## 🛠️ Tech Stack

| Komponen | Detail |
|----------|--------|
| Bahasa | Go 1.22 |
| Database | PostgreSQL 16 |
| CI/CD | GitHub Actions |
| Registry | GitHub Container Registry (GHCR) |
| Containerize | Docker multi-stage (scratch image) |
| Security | gosec (SAST) + gitleaks (Secret Scanning) |
| Notifikasi | Telegram Bot + Slack Webhook |

---

## 🐛 Bug yang Ditemukan & Diperbaiki

| No | File | Bug | Efek |
|----|------|-----|------|
| 1 | `internal/service/service.go` | Integer division pada `CalculateCompletionRate` | Completion rate selalu 0% |
| 2 | `internal/repository/memory.go` + `postgres.go` | `!=` harusnya `==` pada filter status | Filter task mengembalikan hasil terbalik |
| 3 | `internal/validator/validator.go` | `"urgent"` ada di daftar priority valid | Priority invalid lolos validasi |

---

## 🚀 Quick Start

### Prerequisites
- Go 1.22+
- Docker Desktop
- Git

### Clone & Setup
```bash
git clone https://github.com/Nopitrasari29/taskflow-cicd
cd taskflow-cicd/pertemuan-09-cicd/pbl-taskflow-go
```

### Jalankan dengan Docker Compose (paling mudah)
```bash
docker compose up -d
curl http://localhost:8080/health
```

### Jalankan lokal (tanpa Docker)
```bash
cp .env.example .env
# Edit DATABASE_URL di .env jika perlu
go mod tidy
go build -o bin/taskflow-api ./cmd/server
./bin/taskflow-api
```

---

## 🧪 Menjalankan Test

```bash
# Unit test
go test ./internal/validator/... ./internal/service/... ./internal/repository/... -v

# Unit test + race detector
go test -race ./internal/validator/... ./internal/service/... ./internal/repository/...

# Integration test (butuh PostgreSQL aktif)
DATABASE_URL=postgres://taskflow:taskflow_secret@localhost:5432/taskflow?sslmode=disable \
  go test -tags=integration -race ./...

# Coverage report
go test ./... -coverprofile=cov.out
go tool cover -func=cov.out | grep total
```

---

## 🔄 Rollback

Jika terjadi masalah di production, jalankan:
```bash
make rollback ROLLBACK_TAG=sha-xxxxxxx
```

Cari SHA tag lama dari halaman GitHub Actions atau GHCR.

---

## 📁 Struktur Repository

```
taskflow-cicd/
├── .github/
│   └── workflows/
│       └── ci-cd.yml          ← GitHub Actions workflow (5 job)
└── pertemuan-09-cicd/
    └── pbl-taskflow-go/
        ├── cmd/server/        ← Entry point
        ├── internal/
        │   ├── handler/       ← HTTP layer
        │   ├── service/       ← Business logic (Bug #1 ada di sini)
        │   ├── repository/    ← DB layer (Bug #2 ada di sini)
        │   ├── validator/     ← Input validation (Bug #3 ada di sini)
        │   └── model/         ← Struct & types
        ├── migrations/        ← SQL schema
        ├── Dockerfile         ← Multi-stage build
        ├── Makefile           ← Build targets + rollback
        ├── docker-compose.yml ← Local stack
        └── go.mod
```

---

## 📊 Status Pipeline

[![CI](https://github.com/Nopitrasari29/taskflow-cicd/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/Nopitrasari29/taskflow-cicd/actions/workflows/ci-cd.yml)

---

## 📝 Progress per Area

### ✅ Area 1 — Bug Fix + CI Pipeline (Aswalia)
- [x] 3 bug ditemukan dan diperbaiki
- [x] 2 test case baru ditambahkan
- [x] `go test -race` semua PASS
- [x] Coverage ≥ 75% (Coverage Gate PASS)
- [x] 5 job CI pipeline selesai (lint, unit-test, integration-test, coverage-gate, build-binary)

### 🔄 Area 2 — Docker + GHCR (Rafika)
- [x] Dockerfile multi-stage verified
- [x] Image push ke GHCR dengan tag SHA
- [x] Perbandingan ukuran image terdokumentasi

### 🔄 Area 3 — Smoke Test + Rollback (Nisrina)
- [ ] Job smoke-test selesai
- [ ] Notifikasi Telegram + Slack sukses & gagal
- [ ] `make rollback` berfungsi
- [ ] Prosedur rollback terdokumentasi

### 🔄 Area 4 — Security + Laporan (Riskiyatul)
- [ ] gosec (SAST) diintegrasikan ke pipeline
- [ ] gitleaks (Secret Scanning) diintegrasikan
- [ ] Laporan security per kategori
- [ ] Laporan keseluruhan selesai
