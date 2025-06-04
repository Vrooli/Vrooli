

setup_development_environment() {
    header "Installing global dependencies"
    installedPackages=$(yarn global list)
    check_and_add_to_install_list() {
        package="$1"
        version="$2"
        fullPackageName="$package"
        if [ ! -z "$version" ]; then
            fullPackageName="$package@$version"
        fi
        # Check if package is installed globally
        if ! echo "$installedPackages" | grep -E "info \"$package(@$version)?" >/dev/null; then
            info "Installing $fullPackageName"
            toInstall="$toInstall $fullPackageName"
        fi
    }
    toInstall=""
    check_and_add_to_install_list "typescript" "5.3.3"
    check_and_add_to_install_list "nodemon" "3.0.2"
    check_and_add_to_install_list "prisma" "6.1.0"
    check_and_add_to_install_list "vite" "5.2.13"
    # Install all at once if there are packages to install
    if [ ! -z "$toInstall" ]; then
        yarn global add $toInstall
        if [ $? -ne 0 ]; then
            error "Failed to install global dependencies: $toInstall"
            info "Trying to install each package individually..."
            # Split the toInstall string into an array
            IFS=' ' read -r -a individualPackages <<<"$toInstall"
            # Loop through each package and try to install it individually
            for pkg in "${individualPackages[@]}"; do
                info "Attempting to install $pkg individually..."
                yarn global add "$pkg"
                if [ $? -ne 0 ]; then
                    error "Failed to install $pkg"
                    cd "$ORIGINAL_DIR"
                    exit 1
                else
                    info "$pkg installed successfully"
                fi
            done
        fi
    fi

    "${HERE}/shared.sh"

    # Install AWS CLI, for uploading to S3 bucket. This is used for Kubernetes deployments.
    sudo DEBIAN_FRONTEND=noninteractive apt-get install -y awscli
}
