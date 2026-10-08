package planproto

type enumPair[Model comparable, Proto comparable] struct {
	model Model
	proto Proto
}

func enumToProto[Model comparable, Proto comparable](value Model, pairs []enumPair[Model, Proto], fallback Proto) Proto {
	for _, pair := range pairs {
		if pair.model == value {
			return pair.proto
		}
	}
	return fallback
}

func enumFromProto[Model comparable, Proto comparable](value Proto, pairs []enumPair[Model, Proto], fallback Model) Model {
	for _, pair := range pairs {
		if pair.proto == value {
			return pair.model
		}
	}
	return fallback
}
