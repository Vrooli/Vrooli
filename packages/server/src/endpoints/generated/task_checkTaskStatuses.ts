export const task_checkTaskStatuses = {
  "fieldName": "checkTaskStatuses",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "checkTaskStatuses",
        "loc": {
          "start": 65,
          "end": 82
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 83,
              "end": 88
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 91,
                "end": 96
              }
            },
            "loc": {
              "start": 90,
              "end": 96
            }
          },
          "loc": {
            "start": 83,
            "end": 96
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
              "value": "statuses",
              "loc": {
                "start": 104,
                "end": 112
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 123,
                      "end": 125
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 123,
                    "end": 125
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "status",
                    "loc": {
                      "start": 134,
                      "end": 140
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 134,
                    "end": 140
                  }
                }
              ],
              "loc": {
                "start": 113,
                "end": 146
              }
            },
            "loc": {
              "start": 104,
              "end": 146
            }
          }
        ],
        "loc": {
          "start": 98,
          "end": 150
        }
      },
      "loc": {
        "start": 65,
        "end": 150
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
      "value": "checkTaskStatuses",
      "loc": {
        "start": 10,
        "end": 27
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
              "start": 29,
              "end": 34
            }
          },
          "loc": {
            "start": 28,
            "end": 34
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "CheckTaskStatusesInput",
              "loc": {
                "start": 36,
                "end": 58
              }
            },
            "loc": {
              "start": 36,
              "end": 58
            }
          },
          "loc": {
            "start": 36,
            "end": 59
          }
        },
        "directives": [],
        "loc": {
          "start": 28,
          "end": 59
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
            "value": "checkTaskStatuses",
            "loc": {
              "start": 65,
              "end": 82
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 83,
                  "end": 88
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 91,
                    "end": 96
                  }
                },
                "loc": {
                  "start": 90,
                  "end": 96
                }
              },
              "loc": {
                "start": 83,
                "end": 96
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
                  "value": "statuses",
                  "loc": {
                    "start": 104,
                    "end": 112
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "id",
                        "loc": {
                          "start": 123,
                          "end": 125
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 123,
                        "end": 125
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "status",
                        "loc": {
                          "start": 134,
                          "end": 140
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 134,
                        "end": 140
                      }
                    }
                  ],
                  "loc": {
                    "start": 113,
                    "end": 146
                  }
                },
                "loc": {
                  "start": 104,
                  "end": 146
                }
              }
            ],
            "loc": {
              "start": 98,
              "end": 150
            }
          },
          "loc": {
            "start": 65,
            "end": 150
          }
        }
      ],
      "loc": {
        "start": 61,
        "end": 152
      }
    },
    "loc": {
      "start": 1,
      "end": 152
    }
  },
  "variableValues": {},
  "path": {
    "key": "task_checkTaskStatuses"
  }
} as const;
