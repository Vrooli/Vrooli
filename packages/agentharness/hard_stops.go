package agentharness

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
)

// hard_stops.json is the single source of the agent-guard policy: what each
// guard protects, what an agent is told when it stops, and the vocabulary that
// recognizes money, credentials, outbound messages and hidden commands. The Go
// code applies it; it holds no word lists of its own.
//
//go:embed hard_stops.json
var hardStopsJSON []byte

// guardInfo is one guard as the agent, the operator and the steward see it.
type guardInfo struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Why     string   `json:"why"`
	Instead []string `json:"instead"`
}

type hardStopVocabulary struct {
	About  string `json:"about"`
	Guards struct {
		About          string    `json:"about"`
		Money          guardInfo `json:"money"`
		Secret         guardInfo `json:"secret"`
		OutsideMessage guardInfo `json:"outside_message"`
		HiddenEffect   guardInfo `json:"hidden_effect"`
	} `json:"guards"`
	Shell struct {
		About                string   `json:"about"`
		Tools                []string `json:"tools"`
		ShellPrograms        []string `json:"shell_programs"`
		Wrappers             []string `json:"wrappers"`
		Interpreters         []string `json:"interpreters"`
		InterpreterCodeFlags []string `json:"interpreter_code_flags"`
		RemoteShellCarriers  []string `json:"remote_shell_carriers"`
		Readers              []string `json:"readers"`
		EnvironmentReaders   []string `json:"environment_readers"`
	} `json:"shell"`
	EffectTools struct {
		About string   `json:"about"`
		Words []string `json:"words"`
		Calls []string `json:"calls"`
	} `json:"effect_tools"`
	Money struct {
		About           string   `json:"about"`
		Words           []string `json:"words"`
		CheckoutWords   []string `json:"checkout_words"`
		CheckoutContext []string `json:"checkout_context"`
		Calls           []string `json:"calls"`
	} `json:"money"`
	Secrets struct {
		About          string   `json:"about"`
		Tools          []string `json:"tools"`
		ToolOperations []string `json:"tool_operations"`
		Operations     []string `json:"operations"`
		Resources      []string `json:"resources"`
		FilePaths      []string `json:"file_paths"`
		FileTools      []string `json:"file_tools"`
		Calls          []string `json:"calls"`
	} `json:"secrets"`
	Messages struct {
		About             string   `json:"about"`
		Transports        []string `json:"transports"`
		Verbs             []string `json:"verbs"`
		Commands          []string `json:"commands"`
		Calls             []string `json:"calls"`
		WebClients        []string `json:"web_clients"`
		WebClientPrograms []string `json:"web_client_programs"`
		WriteMethods      []string `json:"write_methods"`
		BodyFlags         []string `json:"body_flags"`
		BodyFlagPrefixes  []string `json:"body_flag_prefixes"`
		CommentMutations  []string `json:"comment_mutations"`
		CommentObjects    []string `json:"comment_objects"`
		CommandSequences  []string `json:"command_sequences"`
		MessageHosts      []string `json:"message_hosts"`
		CommentProviders  []string `json:"comment_providers"`
	} `json:"messages"`
	HiddenEffect struct {
		About               string   `json:"about"`
		Decoders            []string `json:"decoders"`
		EncodedCommandFlags []string `json:"encoded_command_flags"`
	} `json:"hidden_effect"`
}

// hardStops is the loaded policy. A policy file that does not load leaves it
// empty and records hardStopsErr: every guard then matches nothing and each
// decision says the policy is unavailable. A broken data file must never
// crash the hook, because a crashed hook blocks every command of every agent.
var hardStops, hardStopsErr = loadHardStops(hardStopsJSON)

// loadHardStops refuses unknown keys and any empty list or description, so a
// typo in the data file fails loudly instead of silently matching nothing.
func loadHardStops(data []byte) (hardStopVocabulary, error) {
	var vocabulary hardStopVocabulary
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&vocabulary); err != nil {
		return hardStopVocabulary{}, fmt.Errorf("hard_stops.json: %w", err)
	}
	if err := requireFilled(reflect.ValueOf(vocabulary), "hard_stops"); err != nil {
		return hardStopVocabulary{}, err
	}
	return vocabulary, nil
}

func requireFilled(value reflect.Value, path string) error {
	switch value.Kind() {
	case reflect.Struct:
		for index := 0; index < value.NumField(); index++ {
			field := value.Type().Field(index)
			if err := requireFilled(value.Field(index), path+"."+field.Tag.Get("json")); err != nil {
				return err
			}
		}
	case reflect.String, reflect.Slice:
		if value.Len() == 0 {
			return fmt.Errorf("hard_stops.json: %s is empty", path)
		}
	}
	return nil
}

// guardFor returns the guard a hard-stop class belongs to.
func guardFor(class HardStopClass) guardInfo {
	switch class {
	case HardStopMoney:
		return hardStops.Guards.Money
	case HardStopSecret:
		return hardStops.Guards.Secret
	case HardStopOutsideMessage:
		return hardStops.Guards.OutsideMessage
	default:
		return hardStops.Guards.HiddenEffect
	}
}
