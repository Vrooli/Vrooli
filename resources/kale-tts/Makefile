.PHONY: validate test check

validate:
	jq empty resource.json
	python3 -m json.tool resource.json >/dev/null
	vrooli resource validate kale-tts

test:
	PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests -v

check: validate test
