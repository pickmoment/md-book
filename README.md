# md-book

마크다운 파일 디렉토리를 책처럼 렌더링해서 브라우저로 보여주는 로컬 CLI 도구.

Go `embed`로 HTML/CSS/JS를 바이너리에 내장하기 때문에 `go build` 한 번으로 외부 의존성 없는 단일 실행 파일이 만들어진다.

## 설치

```bash
go install github.com/pickmoment/md-book@latest
```

또는 직접 빌드:

```bash
git clone https://github.com/pickmoment/md-book
cd md-book
go build -o md-book .
```

## 사용법

```bash
md-book serve <directory> [--port N] [--no-open]
```

| 옵션 | 기본값 | 설명 |
|------|--------|------|
| `--port` | `3000` | 수신 포트 |
| `--no-open` | `false` | 브라우저 자동 열기 비활성화 |

```bash
# 현재 디렉토리를 브라우저로 열기
md-book serve .

# 포트 변경
md-book serve ./docs --port 8080

# 브라우저 자동 열기 없이 서버만 실행
md-book serve ./docs --no-open
```

## 기능

- **Table of Contents** — 디렉토리 구조를 트리로 탐색하는 사이드바
- **Live Reload** — 파일 변경 시 브라우저 자동 새로고침 (`fsnotify` + SSE)
- **Reading Controls** — 글자 크기(14–24px)와 본문 폭(좁게/보통/넓게) 조절. `localStorage`에 저장
- **Syntax Highlighting** — 코드 블록 하이라이팅 (`chroma`)
- **Page Title 자동 추출** — frontmatter `title` → 첫 번째 `# H1` → 파일명 순 폴백
- **CJK 지원** — 한글 등 멀티바이트 문자 안전 처리

## 디렉토리 구조와 URL

파일 경로가 URL로 그대로 매핑된다. `.md` 확장자만 제거되며, 숫자 접두사(`NN-`)는 URL에 그대로 유지된다.

```
docs/
├── 01-intro/
│   ├── README.md       →  /01-intro
│   └── 01-overview.md  →  /01-intro/01-overview
└── 02-basics/
    └── 01-variables.md →  /02-basics/01-variables
```

- `index.md` 또는 `README.md`가 있는 디렉토리는 클릭 가능한 **Chapter**
- 순서는 파일명 알파벳 순. 숫자 접두사(`01-`, `02-`)로 직접 제어 가능

## Manifest (`book.toml`)

Book 루트에 `book.toml`을 두면 순서와 제목을 명시적으로 지정할 수 있다.

```toml
title = "My Book"

[[pages]]
path = "intro"
title = "Introduction"

[[pages]]
path = "02-basics"
```

Manifest가 없으면 알파벳 순으로 자동 탐색한다.

## 의존성

| 패키지 | 용도 |
|--------|------|
| `yuin/goldmark` | 마크다운 렌더링 |
| `yuin/goldmark-meta` | frontmatter 파싱 |
| `alecthomas/chroma` | 코드 하이라이팅 |
| `fsnotify/fsnotify` | 파일 변경 감지 |
| `BurntSushi/toml` | Manifest 파싱 |
