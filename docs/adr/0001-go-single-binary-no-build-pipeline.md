# Go + embed로 외부 빌드 파이프라인 없는 단일 바이너리

로컬 CLI 도구이므로 사용자가 설치 없이 바이너리 하나로 바로 실행할 수 있어야 한다는 요구가 있었다. Node.js(npx)나 Python(pip)도 고려했지만, Go의 `embed` 패키지를 사용하면 HTML/CSS/JS 프론트엔드 자산까지 바이너리에 내장할 수 있어 `go build` 한 번으로 외부 의존성 없는 단일 실행 파일이 만들어진다. 그 결과 Node/npm 생태계(marked, remark 등 마크다운 라이브러리)를 포기했지만, Go의 `goldmark`가 충분한 확장성을 제공하고 배포 단순성이 그 트레이드오프를 정당화한다.

## Considered Options

- **Node.js + npx** — 마크다운 생태계가 풍부하고 `npx md-book` 한 줄 실행 가능. 단, Node.js 런타임이 설치되어 있어야 하고, 프론트엔드 빌드 파이프라인(bundler 등)이 추가될 가능성이 높다.
- **Python + pip** — mkdocs 같은 선례가 있으나, 가상환경 관리 등 설치 경험이 일관되지 않다.
- **Go + embed** — 채택. 런타임 불필요, 단일 바이너리, `go build`만으로 프론트엔드 포함 전체 빌드.
