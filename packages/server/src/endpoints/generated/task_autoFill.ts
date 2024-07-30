export const task_autoFill = {
  "fieldName": "autoFill",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "autoFill",
        "loc": {
          "start": 47,
          "end": 55
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 56,
              "end": 61
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 64,
                "end": 69
              }
            },
            "loc": {
              "start": 63,
              "end": 69
            }
          },
          "loc": {
            "start": 56,
            "end": 69
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
              "value": "data",
              "loc": {
                "start": 77,
                "end": 81
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 77,
              "end": 81
            }
          }
        ],
        "loc": {
          "start": 71,
          "end": 85
        }
      },
      "loc": {
        "start": 47,
        "end": 85
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
      "value": "autoFill",
      "loc": {
        "start": 10,
        "end": 18
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
              "start": 20,
              "end": 25
            }
          },
          "loc": {
            "start": 19,
            "end": 25
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "AutoFillInput",
              "loc": {
                "start": 27,
                "end": 40
              }
            },
            "loc": {
              "start": 27,
              "end": 40
            }
          },
          "loc": {
            "start": 27,
            "end": 41
          }
        },
        "directives": [],
        "loc": {
          "start": 19,
          "end": 41
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
            "value": "autoFill",
            "loc": {
              "start": 47,
              "end": 55
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 56,
                  "end": 61
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 64,
                    "end": 69
                  }
                },
                "loc": {
                  "start": 63,
                  "end": 69
                }
              },
              "loc": {
                "start": 56,
                "end": 69
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
                  "value": "data",
                  "loc": {
                    "start": 77,
                    "end": 81
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 77,
                  "end": 81
                }
              }
            ],
            "loc": {
              "start": 71,
              "end": 85
            }
          },
          "loc": {
            "start": 47,
            "end": 85
          }
        }
      ],
      "loc": {
        "start": 43,
        "end": 87
      }
    },
    "loc": {
      "start": 1,
      "end": 87
    }
  },
  "variableValues": {},
  "path": {
    "key": "task_autoFill"
  }
} as const;
