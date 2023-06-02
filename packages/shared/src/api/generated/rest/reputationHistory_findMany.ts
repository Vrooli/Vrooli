export const reputationHistory_findMany = {
  "fieldName": "reputationHistories",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reputationHistories",
        "loc": {
          "start": 70,
          "end": 89
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 90,
              "end": 95
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 98,
                "end": 103
              }
            },
            "loc": {
              "start": 97,
              "end": 103
            }
          },
          "loc": {
            "start": 90,
            "end": 103
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
              "value": "edges",
              "loc": {
                "start": 111,
                "end": 116
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
                    "value": "cursor",
                    "loc": {
                      "start": 127,
                      "end": 133
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 127,
                    "end": 133
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 142,
                      "end": 146
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
                            "start": 161,
                            "end": 163
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 161,
                          "end": 163
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 176,
                            "end": 186
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 176,
                          "end": 186
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 199,
                            "end": 209
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 199,
                          "end": 209
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "amount",
                          "loc": {
                            "start": 222,
                            "end": 228
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 222,
                          "end": 228
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "event",
                          "loc": {
                            "start": 241,
                            "end": 246
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 241,
                          "end": 246
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "objectId1",
                          "loc": {
                            "start": 259,
                            "end": 268
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 259,
                          "end": 268
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "objectId2",
                          "loc": {
                            "start": 281,
                            "end": 290
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 281,
                          "end": 290
                        }
                      }
                    ],
                    "loc": {
                      "start": 147,
                      "end": 300
                    }
                  },
                  "loc": {
                    "start": 142,
                    "end": 300
                  }
                }
              ],
              "loc": {
                "start": 117,
                "end": 306
              }
            },
            "loc": {
              "start": 111,
              "end": 306
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 311,
                "end": 319
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
                    "value": "endCursor",
                    "loc": {
                      "start": 330,
                      "end": 339
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 330,
                    "end": 339
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 348,
                      "end": 359
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 348,
                    "end": 359
                  }
                }
              ],
              "loc": {
                "start": 320,
                "end": 365
              }
            },
            "loc": {
              "start": 311,
              "end": 365
            }
          }
        ],
        "loc": {
          "start": 105,
          "end": 369
        }
      },
      "loc": {
        "start": 70,
        "end": 369
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
      "value": "reputationHistories",
      "loc": {
        "start": 7,
        "end": 26
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
              "start": 28,
              "end": 33
            }
          },
          "loc": {
            "start": 27,
            "end": 33
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "ReputationHistorySearchInput",
              "loc": {
                "start": 35,
                "end": 63
              }
            },
            "loc": {
              "start": 35,
              "end": 63
            }
          },
          "loc": {
            "start": 35,
            "end": 64
          }
        },
        "directives": [],
        "loc": {
          "start": 27,
          "end": 64
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
            "value": "reputationHistories",
            "loc": {
              "start": 70,
              "end": 89
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 90,
                  "end": 95
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 98,
                    "end": 103
                  }
                },
                "loc": {
                  "start": 97,
                  "end": 103
                }
              },
              "loc": {
                "start": 90,
                "end": 103
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
                  "value": "edges",
                  "loc": {
                    "start": 111,
                    "end": 116
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
                        "value": "cursor",
                        "loc": {
                          "start": 127,
                          "end": 133
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 127,
                        "end": 133
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 142,
                          "end": 146
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
                                "start": 161,
                                "end": 163
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 161,
                              "end": 163
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "created_at",
                              "loc": {
                                "start": 176,
                                "end": 186
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 176,
                              "end": 186
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "updated_at",
                              "loc": {
                                "start": 199,
                                "end": 209
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 199,
                              "end": 209
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "amount",
                              "loc": {
                                "start": 222,
                                "end": 228
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 222,
                              "end": 228
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "event",
                              "loc": {
                                "start": 241,
                                "end": 246
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 241,
                              "end": 246
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "objectId1",
                              "loc": {
                                "start": 259,
                                "end": 268
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 259,
                              "end": 268
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "objectId2",
                              "loc": {
                                "start": 281,
                                "end": 290
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 281,
                              "end": 290
                            }
                          }
                        ],
                        "loc": {
                          "start": 147,
                          "end": 300
                        }
                      },
                      "loc": {
                        "start": 142,
                        "end": 300
                      }
                    }
                  ],
                  "loc": {
                    "start": 117,
                    "end": 306
                  }
                },
                "loc": {
                  "start": 111,
                  "end": 306
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 311,
                    "end": 319
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
                        "value": "endCursor",
                        "loc": {
                          "start": 330,
                          "end": 339
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 330,
                        "end": 339
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 348,
                          "end": 359
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 348,
                        "end": 359
                      }
                    }
                  ],
                  "loc": {
                    "start": 320,
                    "end": 365
                  }
                },
                "loc": {
                  "start": 311,
                  "end": 365
                }
              }
            ],
            "loc": {
              "start": 105,
              "end": 369
            }
          },
          "loc": {
            "start": 70,
            "end": 369
          }
        }
      ],
      "loc": {
        "start": 66,
        "end": 371
      }
    },
    "loc": {
      "start": 1,
      "end": 371
    }
  },
  "variableValues": {},
  "path": {
    "key": "reputationHistory_findMany"
  }
};
