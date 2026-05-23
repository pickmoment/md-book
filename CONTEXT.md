# md-book

로컬 CLI 도구로, 마크다운 파일들이 있는 디렉토리를 책처럼 렌더링해서 브라우저로 보여주는 웹서버.

## Language

**Book**:
`md-book serve`에 넘기는 최상위 디렉토리 전체. 하나의 독립된 읽기 단위.
_Avoid_: 프로젝트, 사이트, 문서

**Page**:
하나의 `.md` 파일이 렌더링된 결과. Book 안의 최소 읽기 단위.
_Avoid_: 문서, 파일, 포스트

**Chapter**:
`index.md` 또는 `README.md`를 포함한 디렉토리. 그 자체가 클릭 가능한 Page이며, 하위 Page들을 그룹핑하는 단위.
_Avoid_: 섹션, 폴더, 디렉토리

**Table of Contents (ToC)**:
Book의 전체 계층 구조를 보여주는 사이드바 내비게이션. Chapter와 Page의 트리 구조를 순서대로 나열.
_Avoid_: 목차, 사이드바, 네비게이션

**Manifest**:
Book 루트의 설정 파일 (`book.toml`). Page/Chapter의 명시적 순서와 메타데이터를 정의. 없으면 알파벳 순으로 폴백.
_Avoid_: 설정 파일, config, toc 파일

**Page Title**:
ToC와 브라우저 탭에 표시되는 Page의 이름. frontmatter `title` → 첫 번째 `# H1` → 파일명 정리 순으로 결정.
_Avoid_: 제목, 이름, 헤딩

**Live Reload**:
파일 변경 감지 시 브라우저를 자동으로 새로고침하는 기능. `fsnotify` + SSE로 구현.
_Avoid_: 핫 리로드, 자동 새로고침, watch mode

**URL Path**:
Page의 브라우저 주소. 파일의 Book 루트 기준 상대 경로에서 `.md`만 제거한 형태. 숫자 접두사(`NN-`)는 그대로 유지. `index.md`/`README.md`는 상위 디렉토리 경로로 매핑. 예: `02-basics/01-variables.md` → `/02-basics/01-variables`.
_Avoid_: 파일 경로, slug

**Filename Title**:
frontmatter도 H1도 없을 때 파일명에서 파생하는 Page Title. 숫자 접두사(`NN-`) 제거 → `-`/`_`를 공백으로 대체 → Title Case 적용. Title Case는 첫 rune 기준으로 처리하므로 한글 등 멀티바이트 문자도 안전.
_Avoid_: 자동 제목, 기본 제목

**Reading Controls**:
사이드바 상단에 위치하는 뷰 조정 컨트롤. 글자 크기(A−/A+, 14–24px, 1px 단위)와 본문 폭(좁게 28rem / 보통 38rem / 넓게 52rem) 두 가지. 설정값은 `localStorage`에 저장되어 페이지 이동 후에도 유지.
_Avoid_: 설정, 옵션, 환경설정

## Example dialogue

> "이 디렉토리에 `README.md`가 없는데 Chapter로 봐야 해?"
> "아니, `index.md`나 `README.md` 둘 다 없으면 Chapter가 아니야 — ToC에서 클릭할 수 없는 그룹 헤더로만 표시돼."

> "Manifest 없이 쓰면 순서가 보장돼?"
> "파일시스템 알파벳 순이야. `01-`, `02-` 같은 숫자 접두사로 네가 순서를 직접 관리하거나, Manifest로 명시하면 돼. URL에도 숫자 접두사가 그대로 남아서 `/02-basics/01-variables` 이런 식이야."

> "글자 크기 바꾸면 다음에 다시 열어도 유지돼?"
> "응, Reading Controls 설정은 localStorage에 저장돼. 폭 설정도 마찬가지야."

> "Page Title이 H1이랑 frontmatter 둘 다 있으면?"
> "frontmatter `title`이 이겨. H1은 페이지 본문에만 표시되고, ToC는 frontmatter 값을 써."
