<h1 align="center" style="border-bottom: none">
    <a target="_blank"><img alt="opsbridge" src="docs/logo.png"></a><br>OpsBridge
</h1>

**OpsBridge** is a high-performance Go-based bridge designed to eliminate context-switching during incident response. It monitors Mattermost alert threads and automatically links them to the corresponding tickets in Gitea using case-insensitive label-matching logic.

## 🚀 Features

* **Smart Ticket Matching**: Bypasses unreliable search indexing by using case-insensitive memory scanning for `alertname` and `certname` labels.
* **Live Status Reports**: Instantly fetch ticket state (🟢 Open / 🔴 Closed), active assignees, and filtered alert labels directly in the thread.
* **Bi-Directional Sync**:
    * `/ticket [comment]`: Push replies from Mattermost directly to Gitea as issue comments.
    * `/assignme` / `/assign @user`: Sync ticket ownership instantly across platforms (assumes matching usernames).
* **Noise Reduction**: Automatically filters out administrative labels like `Incident` or `Time To Handle` to keep chat threads focused.
* **Proxy Support**: Built to handle Gitea instances sitting behind Basic Auth security proxies.

## 🛠 Tech Stack

* **Language**: Go (Golang)
* **Communication**: Mattermost WebSocket API (v6) & Gitea REST API
* **Cloud Infrastructure**: Optimized for Azure (AKS) and AWS

## 📖 Usage

Once an alert is posted in Mattermost, reply to the thread with the following commands:

1.  **Get Status**: Type `/ticket` to fetch the Gitea link, status, and labels.
2.  **Add Comment**: Type `/ticket [Your comment here]` to push a note to Gitea.
3.  **Self-Assignment**: Type `/assignme` to take ownership of the ticket.
4.  **Assign Colleague**: Type `/assign @username` (e.g., `/assign @faizan`).

## 🏗 Configuration

The application is configured via environment variables:

| Variable | Description |
| :--- | :--- |
| `GITEA_URL` | Base URL of your Gitea instance |
| `GITEA_TOKEN` | Gitea Personal Access Token |
| `GITEA_OWNER` | Gitea Organization or User name |
| `GITEA_REPO` | Target Gitea Repository |
| `MATTERMOST_URL` | Base URL of your Mattermost server |
| `MATTERMOST_BOT_TOKEN` | Mattermost Bot Account Token |
| `GITEA_BASIC_USER` | (Optional) Username for Basic Auth proxy |
| `GITEA_BASIC_PASS` | (Optional) Password for Basic Auth proxy |

## 🐳 Deployment

The project includes a multi-stage `Dockerfile` for minimal image size and is ready for deployment as a Kubernetes `Deployment`.

---
*Created by [Divyam](https://github.com/nihaldivyam)*
