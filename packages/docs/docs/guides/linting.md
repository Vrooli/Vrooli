# Linting
Linting is the process that underlines errors and warnings in the code. It is a great way to catch errors before they happen, and to enforce code style. We use the [ESLINT VSCode extension](https://marketplace.visualstudio.com/items?itemName=dbaeumer.vscode-eslint) for linting.

## ESLint
ESLint is configured with an `.eslintrc` file. We have a main `.eslintrc` file in the root of the project, and a `.eslintrc` file in any package that needs to add to the main configuration.

## Manual Linting
Since linting can be computationally expensive, the linter only works on opened files. If you want to lint the entire project:  
1. Enter `CTRL+SHIFT+P` to open the Command Palette  
2. Select `Tasks: Run Task`  
3. Select `eslint: lint whole folder`

## Disabling the Linter
If the linter is taking up too many resources and affecting your workflow, you can temporarily disable it in VSCode:
1. Open the Command Palette with `CTRL+SHIFT+P`.
2. Type and select `Extensions: Focus on Extensions View`.
3. Search for "ESLint" in the list.
4. Click the gear icon next to the ESLint extension and choose `Disable`.

To re-enable the linter, follow the same steps and choose `Enable` instead.