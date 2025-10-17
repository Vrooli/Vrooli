#!/usr/bin/env bash
################################################################################
# SageMath Mathematical Operations Library
# 
# Extended mathematical operations for common use cases
################################################################################

# Solve equations
sagemath::math::solve() {
    local equation="${1:-}"
    local variable="${2:-x}"
    
    if [ -z "$equation" ]; then
        echo "Usage: resource-sagemath math solve \"equation\" [variable]"
        echo "Example: resource-sagemath math solve \"x^2 - 4 == 0\" x"
        return 1
    fi
    
    sagemath::content::calculate "solve($equation, $variable)"
}

# Differentiate expressions
sagemath::math::differentiate() {
    local expression="${1:-}"
    local variable="${2:-x}"
    local order="${3:-1}"
    
    if [ -z "$expression" ]; then
        echo "Usage: resource-sagemath math differentiate \"expression\" [variable] [order]"
        echo "Example: resource-sagemath math differentiate \"sin(x)\" x 2"
        return 1
    fi
    
    sagemath::content::calculate "diff($expression, $variable, $order)"
}

# Integrate expressions
sagemath::math::integrate() {
    local expression="${1:-}"
    local variable="${2:-x}"
    local lower="${3:-}"
    local upper="${4:-}"
    
    if [ -z "$expression" ]; then
        echo "Usage: resource-sagemath math integrate \"expression\" [variable] [lower] [upper]"
        echo "Example: resource-sagemath math integrate \"x^2\" x 0 1"
        return 1
    fi
    
    if [ -n "$lower" ] && [ -n "$upper" ]; then
        sagemath::content::calculate "integrate($expression, $variable, $lower, $upper)"
    else
        sagemath::content::calculate "integrate($expression, $variable)"
    fi
}

# Matrix operations
sagemath::math::matrix() {
    local operation="${1:-}"
    local matrix_data="${2:-}"
    
    case "$operation" in
        determinant|det)
            sagemath::content::calculate "matrix($matrix_data).det()"
            ;;
        inverse|inv)
            sagemath::content::calculate "matrix($matrix_data).inverse()"
            ;;
        eigenvalues|eig)
            sagemath::content::calculate "matrix($matrix_data).eigenvalues()"
            ;;
        rank)
            sagemath::content::calculate "matrix($matrix_data).rank()"
            ;;
        transpose|T)
            sagemath::content::calculate "matrix($matrix_data).transpose()"
            ;;
        *)
            echo "Usage: resource-sagemath math matrix <operation> \"[[matrix_data]]\"" 
            echo "Operations: determinant, inverse, eigenvalues, rank, transpose"
            echo "Example: resource-sagemath math matrix det \"[[1,2],[3,4]]\""
            return 1
            ;;
    esac
}

# Number theory operations
sagemath::math::prime() {
    local operation="${1:-}"
    local number="${2:-}"
    
    case "$operation" in
        check|is)
            sagemath::content::calculate "is_prime($number)"
            ;;
        next)
            sagemath::content::calculate "next_prime($number)"
            ;;
        prev|previous)
            sagemath::content::calculate "previous_prime($number)"
            ;;
        count)
            sagemath::content::calculate "prime_pi($number)"
            ;;
        factor)
            sagemath::content::calculate "factor($number)"
            ;;
        *)
            echo "Usage: resource-sagemath math prime <operation> <number>"
            echo "Operations: check, next, previous, count, factor"
            echo "Example: resource-sagemath math prime check 17"
            return 1
            ;;
    esac
}

# Statistical operations
sagemath::math::stats() {
    local operation="${1:-}"
    local data="${2:-}"
    
    case "$operation" in
        mean|average)
            sagemath::content::calculate "mean($data)"
            ;;
        median)
            sagemath::content::calculate "median($data)"
            ;;
        mode)
            sagemath::content::calculate "mode($data)"
            ;;
        std|stdev)
            sagemath::content::calculate "std($data)"
            ;;
        variance|var)
            sagemath::content::calculate "variance($data)"
            ;;
        *)
            echo "Usage: resource-sagemath math stats <operation> \"[data]\""
            echo "Operations: mean, median, mode, std, variance"
            echo "Example: resource-sagemath math stats mean \"[1,2,3,4,5]\""
            return 1
            ;;
    esac
}

# Polynomial operations
sagemath::math::polynomial() {
    local operation="${1:-}"
    local poly="${2:-}"
    local extra="${3:-}"
    
    case "$operation" in
        expand)
            sagemath::content::calculate "expand($poly)"
            ;;
        factor)
            sagemath::content::calculate "factor($poly)"
            ;;
        roots)
            sagemath::content::calculate "($poly).roots()"
            ;;
        degree)
            sagemath::content::calculate "($poly).degree()"
            ;;
        *)
            echo "Usage: resource-sagemath math polynomial <operation> \"polynomial\""
            echo "Operations: expand, factor, roots, degree"
            echo "Example: resource-sagemath math polynomial expand \"(x+1)^3\""
            return 1
            ;;
    esac
}

# Complex number operations
sagemath::math::complex() {
    local operation="${1:-}"
    local number="${2:-}"
    
    case "$operation" in
        magnitude|abs)
            sagemath::content::calculate "abs($number)"
            ;;
        argument|arg)
            sagemath::content::calculate "arg($number)"
            ;;
        conjugate|conj)
            sagemath::content::calculate "conjugate($number)"
            ;;
        real)
            sagemath::content::calculate "real($number)"
            ;;
        imag|imaginary)
            sagemath::content::calculate "imag($number)"
            ;;
        *)
            echo "Usage: resource-sagemath math complex <operation> \"complex_number\""
            echo "Operations: magnitude, argument, conjugate, real, imaginary"
            echo "Example: resource-sagemath math complex magnitude \"3+4*I\""
            return 1
            ;;
    esac
}

# Limits
sagemath::math::limit() {
    local expression="${1:-}"
    local variable="${2:-}"
    local value="${3:-}"
    local direction="${4:-}"
    
    if [ -z "$expression" ] || [ -z "$variable" ] || [ -z "$value" ]; then
        echo "Usage: resource-sagemath math limit \"expression\" variable value [direction]"
        echo "Example: resource-sagemath math limit \"sin(x)/x\" x 0"
        echo "Example: resource-sagemath math limit \"1/x\" x 0 +"
        return 1
    fi
    
    if [ -n "$direction" ]; then
        sagemath::content::calculate "limit($expression, $variable=$value, dir='$direction')"
    else
        sagemath::content::calculate "limit($expression, $variable=$value)"
    fi
}

# Series expansion
sagemath::math::series() {
    local type="${1:-}"
    local expression="${2:-}"
    local variable="${3:-x}"
    local point="${4:-0}"
    local order="${5:-5}"
    
    case "$type" in
        taylor)
            sagemath::content::calculate "taylor($expression, $variable, $point, $order)"
            ;;
        series)
            sagemath::content::calculate "series($expression, $variable, $point, $order)"
            ;;
        fourier)
            sagemath::content::calculate "($expression).fourier_series_cosine_coefficient(n, $variable)"
            ;;
        *)
            echo "Usage: resource-sagemath math series <type> \"expression\" [variable] [point] [order]"
            echo "Types: taylor, series, fourier"
            echo "Example: resource-sagemath math series taylor \"sin(x)\" x 0 5"
            return 1
            ;;
    esac
}