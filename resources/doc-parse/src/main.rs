use std::io::{self, BufRead, Write};
use std::path::Path;

fn json_string(value: &str) -> String {
    let mut out = String::with_capacity(value.len() + 2);
    out.push('"');
    for ch in value.chars() {
        match ch {
            '"' => out.push_str("\\\""),
            '\\' => out.push_str("\\\\"),
            '\n' => out.push_str("\\n"),
            '\r' => out.push_str("\\r"),
            '\t' => out.push_str("\\t"),
            c if c.is_control() => out.push_str(&format!("\\u{:04x}", c as u32)),
            c => out.push(c),
        }
    }
    out.push('"');
    out
}

fn request_path(line: &str) -> Option<String> {
    let marker = "\"path\"";
    let start = line.find(marker)?;
    let rest = &line[start + marker.len()..];
    let colon = rest.find(':')?;
    let value = rest[colon + 1..].trim_start();
    if !value.starts_with('"') {
        return None;
    }
    let mut escaped = false;
    let mut path = String::new();
    for ch in value[1..].chars() {
        if escaped {
            path.push(match ch {
                'n' => '\n',
                'r' => '\r',
                't' => '\t',
                other => other,
            });
            escaped = false;
        } else if ch == '\\' {
            escaped = true;
        } else if ch == '"' {
            return Some(path);
        } else {
            path.push(ch);
        }
    }
    None
}

fn error_response(path: &str, terminal_state: &str, error: &str) -> String {
    format!(
        "{{\"path\":{},\"terminal_state\":{},\"error\":{}}}",
        json_string(path),
        json_string(terminal_state),
        json_string(error)
    )
}

fn pdf_response(path: &str) -> Result<String, String> {
    let result = pdf_inspector::process_pdf(path).map_err(|err| err.to_string())?;
    let items = pdf_inspector::extract_text_with_positions(path).map_err(|err| err.to_string())?;
    let positioned = items
        .iter()
        .map(|item| {
            format!(
                "{{\"text\":{},\"x\":{},\"y\":{},\"width\":{},\"height\":{},\"font\":{},\"font_size\":{},\"page\":{},\"is_bold\":{},\"is_italic\":{},\"is_underline\":{},\"is_strikeout\":{},\"item_type\":{},\"mcid\":{}}}",
                json_string(&item.text),
                item.x,
                item.y,
                item.width,
                item.height,
                json_string(&item.font),
                item.font_size,
                item.page,
                item.is_bold,
                item.is_italic,
                item.is_underline,
                item.is_strikeout,
                json_string(&format!("{:?}", item.item_type)),
                item.mcid.map_or_else(|| "null".to_owned(), |value| value.to_string())
            )
        })
        .collect::<Vec<_>>()
        .join(",");
    let markdown = result
        .markdown
        .as_deref()
        .map(json_string)
        .unwrap_or_else(|| "null".to_owned());
    Ok(format!(
        "{{\"path\":{},\"format\":\"pdf\",\"terminal_state\":\"parsed\",\"markdown\":{},\"pdf_type\":{},\"confidence\":{},\"page_count\":{},\"processing_time_ms\":{},\"pages_needing_ocr\":{:?},\"has_encoding_issues\":{},\"positioned_text\":[{}]}}",
        json_string(path),
        markdown,
        json_string(&format!("{:?}", result.pdf_type)),
        result.confidence,
        result.page_count,
        0,
        result.pages_needing_ocr,
        result.has_encoding_issues,
        positioned
    ))
}

fn office_response(path: &str) -> Result<String, String> {
    let markdown = anydoc::to_markdown(path).map_err(|err| err.to_string())?;
    Ok(format!(
        "{{\"path\":{},\"format\":\"document\",\"terminal_state\":\"parsed\",\"markdown\":{},\"tables\":[]}}",
        json_string(path),
        json_string(&markdown)
    ))
}

fn process(line: &str) -> String {
    let Some(path) = request_path(line) else {
        return error_response("", "malformed_request", "request must contain a JSON string field named path");
    };
    if !Path::new(&path).is_file() {
        return error_response(&path, "missing_input", "input path is not a regular file");
    }
    let is_pdf = Path::new(&path)
        .extension()
        .and_then(|ext| ext.to_str())
        .is_some_and(|ext| ext.eq_ignore_ascii_case("pdf"));
    let result = if is_pdf { pdf_response(&path) } else { office_response(&path) };
    result.unwrap_or_else(|err| error_response(&path, "parse_failed", &err))
}

fn main() -> io::Result<()> {
    let stdin = io::stdin();
    let mut stdout = io::BufWriter::new(io::stdout().lock());
    for line in stdin.lock().lines() {
        let line = line?;
        writeln!(stdout, "{}", process(&line))?;
        stdout.flush()?;
    }
    Ok(())
}
