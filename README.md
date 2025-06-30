# 🐳 gocker

**gocker** is a minimalist and educational Docker implementation written in Go.  
Its main goal is to demonstrate how containers work *under the hood*, using namespaces, cgroups, chroot, and other Linux kernel features — without relying on the Docker daemon.

> ⚠️ **Warning:** This is an experimental project and should not be used in production. It is intended for learning or as a foundation for low-level container exploration.

---

## Implemented Features

- [x] `Dockerfile` parser with support for:
  - `FROM`
  - `COPY`
  - `ENTRYPOINT`
- [x] Download of public images from Docker Hub
- [x] Automatic resolution of the correct image for `GOOS` and `GOARCH`
- [x] Root filesystem assembly from image layers
- [x] Container execution with:
  - `chroot`, `chdir`, `mount proc`, `exec`
- [x] Process re-execution with `GOCKER_INIT=1` for init process isolation
- [x] Resource isolation with **cgroups v2**:
  - Memory limit: 1 GB
  - CPU limit: 10%
- [x] Modular structure using internal packages:
  - `dockerfile`, `image`, `filesystem`, `container`, `build`

---

## Planned Features (future)

- [ ] Support for additional Dockerfile instructions:
  - `RUN`, `CMD`, `ENV`, `WORKDIR`, `VOLUME`, `ARG`
- [ ] Build layer cache implementation
- [ ] Full namespace support:
  - UTS (hostname)
  - PID (process isolation)
  - NET (virtual network)
  - USER (rootless containers)
- [ ] Support for bind mounts and volumes (`-v`)
- [ ] Virtual network bridge implementation
- [ ] UID/GID mapping (for non-root containers)
- [ ] Docker-like CLI (`gocker build`, `gocker run`, etc.)
- [ ] Automated tests for parsing, build, and execution
- [ ] Metadata generation (like `docker history`)

---

## How to Run

```bash
go run ./cmd/gocker
```

> This will:
> - Read the `Dockerfile` from the current directory
> - Download the base image
> - Mount the root filesystem
> - Copy specified files
> - Run the container using the `ENTRYPOINT`

---

## Example `Dockerfile`

```Dockerfile
FROM node:alpine
COPY index.js /app/index.js
ENTRYPOINT /app/index.js
```

---

## Project Structure

```
gocker/
├── cmd/gocker/         # Main binary
├── internal/
│   ├── build/          # Runner that interprets Dockerfile
│   ├── container/      # Container execution with isolation
│   ├── dockerfile/     # Dockerfile parser
│   ├── filesystem/     # Filesystem extraction and mounting
│   └── image/          # Docker Hub image downloader
```

---

## Educational Purpose

This project aims to demonstrate the following concepts:

- How images are represented by layers
- What a `Dockerfile` actually defines
- How Linux isolates processes (`chroot`, `namespaces`)
- How the kernel controls resource usage (`cgroups`)
- How command execution inside containers works