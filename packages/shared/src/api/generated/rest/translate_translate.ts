export const translate_translate = {
  "fieldName": "translate",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translate",
        "loc": {
          "start": 45,
          "end": 54
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 55,
              "end": 60
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 63,
                "end": 68
              }
            },
            "loc": {
              "start": 62,
              "end": 68
            }
          },
          "loc": {
            "start": 55,
            "end": 68
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
              "value": "fields",
              "loc": {
                "start": 76,
                "end": 82
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 76,
              "end": 82
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 87,
                "end": 95
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 87,
              "end": 95
            }
          }
        ],
        "loc": {
          "start": 70,
          "end": 99
        }
      },
      "loc": {
        "start": 45,
        "end": 99
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
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "translate",
      "loc": {
        "start": 7,
        "end": 16
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
              "start": 18,
              "end": 23
            }
          },
          "loc": {
            "start": 17,
            "end": 23
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "FindByIdInput",
              "loc": {
                "start": 25,
                "end": 38
              }
            },
            "loc": {
              "start": 25,
              "end": 38
            }
          },
          "loc": {
            "start": 25,
            "end": 39
          }
        },
        "directives": [],
        "loc": {
          "start": 17,
          "end": 39
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
            "value": "translate",
            "loc": {
              "start": 45,
              "end": 54
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 55,
                  "end": 60
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 63,
                    "end": 68
                  }
                },
                "loc": {
                  "start": 62,
                  "end": 68
                }
              },
              "loc": {
                "start": 55,
                "end": 68
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
                  "value": "fields",
                  "loc": {
                    "start": 76,
                    "end": 82
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 76,
                  "end": 82
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "language",
                  "loc": {
                    "start": 87,
                    "end": 95
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 87,
                  "end": 95
                }
              }
            ],
            "loc": {
              "start": 70,
              "end": 99
            }
          },
          "loc": {
            "start": 45,
            "end": 99
          }
        }
      ],
      "loc": {
        "start": 41,
        "end": 101
      }
    },
    "loc": {
      "start": 1,
      "end": 101
    }
  },
  "variableValues": {},
  "path": {
    "key": "translate_translate"
  }
} as const;
