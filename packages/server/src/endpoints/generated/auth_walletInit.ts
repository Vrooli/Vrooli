export const auth_walletInit = {
  "fieldName": "walletInit",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "walletInit",
        "loc": {
          "start": 51,
          "end": 61
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 62,
              "end": 67
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 70,
                "end": 75
              }
            },
            "loc": {
              "start": 69,
              "end": 75
            }
          },
          "loc": {
            "start": 62,
            "end": 75
          }
        }
      ],
      "directives": [],
      "loc": {
        "start": 51,
        "end": 76
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {},
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "mutation",
    "name": {
      "kind": "Name",
      "value": "walletInit",
      "loc": {
        "start": 10,
        "end": 20
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 22,
              "end": 27
            }
          },
          "loc": {
            "start": 21,
            "end": 27
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "WalletInitInput",
              "loc": {
                "start": 29,
                "end": 44
              }
            },
            "loc": {
              "start": 29,
              "end": 44
            }
          },
          "loc": {
            "start": 29,
            "end": 45
          }
        },
        "directives": [],
        "loc": {
          "start": 21,
          "end": 45
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "walletInit",
            "loc": {
              "start": 51,
              "end": 61
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 62,
                  "end": 67
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 70,
                    "end": 75
                  }
                },
                "loc": {
                  "start": 69,
                  "end": 75
                }
              },
              "loc": {
                "start": 62,
                "end": 75
              }
            }
          ],
          "directives": [],
          "loc": {
            "start": 51,
            "end": 76
          }
        }
      ],
      "loc": {
        "start": 47,
        "end": 78
      }
    },
    "loc": {
      "start": 1,
      "end": 78
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_walletInit"
  }
} as const;
