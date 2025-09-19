#!/usr/bin/env python3
"""OpenCode AI CLI – terminal-first assistant for Vrooli."""
from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path
from typing import Dict, List, Optional, Tuple

CONFIG_DEFAULTS = {
    "provider": "ollama",
    "chat_model": "llama3.2:3b",
    "completion_model": "qwen2.5-coder:3b",
    "port": 3355,
    "auto_suggest": True,
    "enable_logging": True,
}

LOG_DIR = Path(os.environ.get("OPENCODE_LOG_DIR", Path(os.environ.get("OPENCODE_DATA_DIR", ".")) / "logs"))


def log_event(event: str, metadata: Optional[dict] = None) -> None:
    try:
        LOG_DIR.mkdir(parents=True, exist_ok=True)
        payload = {
            "ts": time.time(),
            "event": event,
            "meta": metadata or {},
        }
        with (LOG_DIR / "opencode.log").open("a", encoding="utf-8") as fh:
            fh.write(json.dumps(payload) + "\n")
    except Exception:  # pragma: no cover - logging must not break execution
        pass


def load_config(path: Path) -> dict:
    if not path.exists():
        return CONFIG_DEFAULTS.copy()
    with path.open("r", encoding="utf-8") as fh:
        data = json.load(fh)
    return {**CONFIG_DEFAULTS, **data}


def save_response(text: str, provider: str, model: str) -> None:
    meta = {
        "provider": provider,
        "model": model,
    }
    log_event("response", {**meta, "length": len(text)})


def run_ollama_list() -> List[str]:
    if not shutil_which("ollama"):
        return []
    try:
        proc = subprocess.run(
            ["ollama", "list"],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=True,
            text=True,
        )
        lines = [line.strip().split()[0] for line in proc.stdout.splitlines()[1:] if line.strip()]
        return lines
    except (subprocess.CalledProcessError, FileNotFoundError):
        return []


def run_openrouter_models(api_key: str) -> List[str]:
    if not api_key:
        return []
    req = urllib.request.Request(
        url="https://openrouter.ai/api/v1/models",
        headers={
            "Authorization": f"Bearer {api_key}",
            "Content-Type": "application/json",
            "HTTP-Referer": "https://vrooli.local",
            "X-Title": "Vrooli",
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            payload = json.load(resp)
        return [item["id"] for item in payload.get("data", []) if "id" in item]
    except Exception:
        return []


def run_cloudflare_models(gateway_url: str, api_token: str) -> List[str]:
    if not gateway_url or not api_token:
        return []
    endpoint = gateway_url.rstrip("/") + "/models"
    req = urllib.request.Request(
        url=endpoint,
        headers={
            "Authorization": f"Bearer {api_token}",
            "Content-Type": "application/json",
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            payload = json.load(resp)
        if isinstance(payload, dict) and "models" in payload:
            return payload["models"]
        if isinstance(payload, list):
            return payload
        return []
    except Exception:
        return []


def load_custom_models(data_dir: Path) -> List[str]:
    file_path = data_dir / "available-models.json"
    if not file_path.exists():
        return []
    try:
        with file_path.open("r", encoding="utf-8") as fh:
            payload = json.load(fh)
        if isinstance(payload, list):
            return [str(item) for item in payload]
    except Exception:
        pass
    return []


def ensure_model(provider: str, config: dict, override: Optional[str]) -> Tuple[str, str]:
    model = override or config.get("chat_model")
    if provider == "completion":
        model = override or config.get("completion_model")
    if not model:
        raise RuntimeError("Model not specified")
    return provider, model


def run_ollama_chat(model: str, prompt: str, system: Optional[str]) -> str:
    cmd = ["ollama", "run", model]
    full_prompt = prompt if not system else f"{system}\n\n{prompt}"
    proc = subprocess.run(
        cmd,
        input=full_prompt,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    if proc.returncode != 0:
        raise RuntimeError(proc.stderr.strip() or "ollama failed")
    return proc.stdout.strip()


def run_openrouter_chat(api_key: str, model: str, prompt: str, system: Optional[str], gateway_url: Optional[str] = None) -> str:
    if not api_key:
        raise RuntimeError("OPENROUTER_API_KEY is not configured")

    url = "https://openrouter.ai/api/v1/chat/completions"
    headers = {
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
        "HTTP-Referer": "https://vrooli.local",
        "X-Title": "Vrooli",
    }
    if gateway_url:
        url = gateway_url.rstrip("/") + "/chat/completions"
        headers = {
            "Authorization": f"Bearer {os.environ.get('CLOUDFLARE_API_TOKEN', '')}",
            "Content-Type": "application/json",
        }

    messages = []
    if system:
        messages.append({"role": "system", "content": system})
    messages.append({"role": "user", "content": prompt})

    payload = json.dumps({
        "model": model,
        "messages": messages,
    }).encode("utf-8")

    req = urllib.request.Request(url=url, data=payload, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            data = json.load(resp)
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", "ignore")
        raise RuntimeError(f"HTTP {exc.code}: {detail}") from exc
    except Exception as exc:  # pragma: no cover
        raise RuntimeError(str(exc)) from exc

    try:
        return data["choices"][0]["message"]["content"].strip()
    except Exception as exc:
        raise RuntimeError(f"Unexpected response: {data}") from exc


def run_cloudflare_chat(model: str, prompt: str, system: Optional[str], gateway_url: str, api_token: str) -> str:
    if not gateway_url or not api_token:
        raise RuntimeError("Cloudflare gateway credentials not configured")

    url = gateway_url.rstrip("/") + "/chat/completions"
    messages = []
    if system:
        messages.append({"role": "system", "content": system})
    messages.append({"role": "user", "content": prompt})

    payload = json.dumps({
        "model": model,
        "messages": messages,
    }).encode("utf-8")

    req = urllib.request.Request(
        url=url,
        data=payload,
        headers={
            "Authorization": f"Bearer {api_token}",
            "Content-Type": "application/json",
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=60) as resp:
            data = json.load(resp)
        return data["result"]["choices"][0]["message"]["content"].strip()
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", "ignore")
        raise RuntimeError(f"HTTP {exc.code}: {detail}") from exc
    except Exception as exc:
        raise RuntimeError(str(exc)) from exc


def shutil_which(name: str) -> Optional[str]:
    from shutil import which

    return which(name)


def models_command(args: argparse.Namespace, config: dict, data_dir: Path) -> int:
    models = {
        "ollama": run_ollama_list(),
        "openrouter": run_openrouter_models(os.environ.get("OPENROUTER_API_KEY", "")),
        "cloudflare": run_cloudflare_models(
            config.get("cloudflare_gateway_url", ""),
            os.environ.get("CLOUDFLARE_API_TOKEN", ""),
        ),
        "custom": load_custom_models(data_dir),
    }

    if args.json:
        print(json.dumps(models, indent=2))
    else:
        for provider, entries in models.items():
            if entries:
                print(f"{provider}: {len(entries)}")
                for entry in entries:
                    print(f"  - {entry}")
            else:
                print(f"{provider}: none")
    return 0


def info_command(config: dict, data_dir: Path) -> int:
    provider = config.get("provider", "unknown")
    if provider == "completion":
        provider = config.get("completion_provider", "unknown")

    models = {
        "ollama": run_ollama_list(),
        "openrouter": [],
        "cloudflare": [],
        "custom": load_custom_models(data_dir),
    }
    if os.environ.get("OPENROUTER_API_KEY"):
        models["openrouter"] = run_openrouter_models(os.environ["OPENROUTER_API_KEY"])
    if config.get("cloudflare_gateway_url") and os.environ.get("CLOUDFLARE_API_TOKEN"):
        models["cloudflare"] = run_cloudflare_models(
            config["cloudflare_gateway_url"], os.environ.get("CLOUDFLARE_API_TOKEN", "")
        )

    info = {
        "provider": provider,
        "chat_model": config.get("chat_model"),
        "completion_model": config.get("completion_model"),
        "models": models,
        "secrets": {
            "OPENROUTER_API_KEY": bool(os.environ.get("OPENROUTER_API_KEY")),
            "CLOUDFLARE_API_TOKEN": bool(os.environ.get("CLOUDFLARE_API_TOKEN")),
        },
    }
    print(json.dumps(info))
    return 0


def chat_command(args: argparse.Namespace, config: dict) -> int:
    provider = args.provider or config.get("provider", "ollama")
    prompt = args.prompt
    if not prompt:
        raise RuntimeError("Prompt is required")

    if provider == "ollama":
        output = run_ollama_chat(args.model or config.get("chat_model"), prompt, args.system)
    elif provider == "openrouter":
        gateway_url = config.get("cloudflare_gateway_url") if args.use_gateway else None
        output = run_openrouter_chat(
            api_key=os.environ.get("OPENROUTER_API_KEY", ""),
            model=args.model or config.get("chat_model"),
            prompt=prompt,
            system=args.system,
            gateway_url=gateway_url,
        )
    elif provider == "cloudflare":
        output = run_cloudflare_chat(
            model=args.model or config.get("chat_model"),
            prompt=prompt,
            system=args.system,
            gateway_url=config.get("cloudflare_gateway_url", ""),
            api_token=os.environ.get("CLOUDFLARE_API_TOKEN", ""),
        )
    else:
        raise RuntimeError(f"Unsupported provider: {provider}")

    save_response(output, provider, args.model or config.get("chat_model"))

    if args.raw:
        print(output)
    else:
        print(output.strip())
    return 0


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="OpenCode AI CLI")
    parser.add_argument("--config", default=os.environ.get("OPENCODE_CONFIG_FILE"), help="Path to config.json")

    subparsers = parser.add_subparsers(dest="command")

    info_p = subparsers.add_parser("info", help="Show configuration summary")
    info_p.set_defaults(func=lambda args, cfg, data_dir: info_command(cfg, data_dir))

    models_p = subparsers.add_parser("models", help="Model operations")
    models_sub = models_p.add_subparsers(dest="subcommand")
    models_list = models_sub.add_parser("list", help="List available models")
    models_list.add_argument("--json", action="store_true", help="Return JSON output")

    def models_list_cb(args, cfg, data_dir):
        return models_command(args, cfg, data_dir)

    models_list.set_defaults(func=models_list_cb)

    chat_p = subparsers.add_parser("chat", help="Send a chat completion request")
    chat_p.add_argument("--prompt", required=True, help="User prompt")
    chat_p.add_argument("--system", help="Optional system prompt")
    chat_p.add_argument("--model", help="Override model")
    chat_p.add_argument("--provider", help="Override provider")
    chat_p.add_argument("--use-gateway", action="store_true", help="Route via Cloudflare gateway if configured")
    chat_p.add_argument("--raw", action="store_true", help="Print raw response")

    def chat_cb(args, cfg, data_dir):
        _ = data_dir
        return chat_command(args, cfg)

    chat_p.set_defaults(func=chat_cb)

    return parser


def main(argv: List[str]) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)

    if not args.command:
        parser.print_help()
        return 1

    config_path = Path(args.config or Path.home() / ".config/opencode/config.json")
    data_dir = config_path.parent
    config = load_config(config_path)

    try:
        return args.func(args, config, data_dir)
    except RuntimeError as exc:
        log_event("error", {"reason": str(exc), "command": args.command})
        print(f"Error: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
