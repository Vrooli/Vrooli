package digest

import "math"

func lnBeta(a, b float64) float64 {
	la, _ := math.Lgamma(a)
	lb, _ := math.Lgamma(b)
	lab, _ := math.Lgamma(a + b)
	return la + lb - lab
}

// ProbabilityBeatsControl returns P(pB > pA) for independent Beta(1,1)
// posteriors. Inputs are successes and trials for B followed by A.
func ProbabilityBeatsControl(sB, nB, sA, nA int64) float64 {
	if nB < 0 || nA < 0 || sB < 0 || sA < 0 || sB > nB || sA > nA {
		return 0
	}
	aA, bA := float64(sA+1), float64(nA-sA+1)
	aB, bB := float64(sB+1), float64(nB-sB+1)
	if aB > 20000 {
		return normalApprox(aA, bA, aB, bB)
	}
	total := 0.0
	for i := 0.0; i < aB; i++ {
		total += math.Exp(lnBeta(aA+i, bA+bB) - math.Log(bB+i) - lnBeta(1+i, bB) - lnBeta(aA, bA))
	}
	if total < 0 {
		return 0
	}
	if total > 1 {
		return 1
	}
	return total
}

func normalApprox(aA, bA, aB, bB float64) float64 {
	meanA, meanB := aA/(aA+bA), aB/(aB+bB)
	varA := aA * bA / ((aA + bA) * (aA + bA) * (aA + bA + 1))
	varB := aB * bB / ((aB + bB) * (aB + bB) * (aB + bB + 1))
	z := (meanB - meanA) / math.Sqrt(varA+varB)
	return 0.5 * (1 + math.Erf(z/math.Sqrt2))
}
