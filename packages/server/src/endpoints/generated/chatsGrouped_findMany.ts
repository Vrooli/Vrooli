export const chatsGrouped_findMany = {
  "fieldName": "chatsGrouped",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "chatsGrouped",
        "loc": {
          "start": 58,
          "end": 70
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 71,
              "end": 76
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 79,
                "end": 84
              }
            },
            "loc": {
              "start": 78,
              "end": 84
            }
          },
          "loc": {
            "start": 71,
            "end": 84
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
                "start": 92,
                "end": 97
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
                      "start": 108,
                      "end": 114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 108,
                    "end": 114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 123,
                      "end": 127
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
                            "start": 142,
                            "end": 144
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 142,
                          "end": 144
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "chatsCount",
                          "loc": {
                            "start": 157,
                            "end": 167
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 157,
                          "end": 167
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "user",
                          "loc": {
                            "start": 180,
                            "end": 184
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
                                  "start": 203,
                                  "end": 205
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 203,
                                "end": 205
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bannerImage",
                                "loc": {
                                  "start": 222,
                                  "end": 233
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 222,
                                "end": 233
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "handle",
                                "loc": {
                                  "start": 250,
                                  "end": 256
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 250,
                                "end": 256
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isBot",
                                "loc": {
                                  "start": 273,
                                  "end": 278
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 273,
                                "end": 278
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 295,
                                  "end": 299
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 295,
                                "end": 299
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "profileImage",
                                "loc": {
                                  "start": 316,
                                  "end": 328
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 316,
                                "end": 328
                              }
                            }
                          ],
                          "loc": {
                            "start": 185,
                            "end": 342
                          }
                        },
                        "loc": {
                          "start": 180,
                          "end": 342
                        }
                      }
                    ],
                    "loc": {
                      "start": 128,
                      "end": 352
                    }
                  },
                  "loc": {
                    "start": 123,
                    "end": 352
                  }
                }
              ],
              "loc": {
                "start": 98,
                "end": 358
              }
            },
            "loc": {
              "start": 92,
              "end": 358
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 363,
                "end": 371
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
                      "start": 382,
                      "end": 391
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 382,
                    "end": 391
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasNextPage",
                    "loc": {
                      "start": 400,
                      "end": 411
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 400,
                    "end": 411
                  }
                }
              ],
              "loc": {
                "start": 372,
                "end": 417
              }
            },
            "loc": {
              "start": 363,
              "end": 417
            }
          }
        ],
        "loc": {
          "start": 86,
          "end": 421
        }
      },
      "loc": {
        "start": 58,
        "end": 421
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
      "value": "chatsGrouped",
      "loc": {
        "start": 7,
        "end": 19
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
              "start": 21,
              "end": 26
            }
          },
          "loc": {
            "start": 20,
            "end": 26
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "ChatsGroupedSearchInput",
              "loc": {
                "start": 28,
                "end": 51
              }
            },
            "loc": {
              "start": 28,
              "end": 51
            }
          },
          "loc": {
            "start": 28,
            "end": 52
          }
        },
        "directives": [],
        "loc": {
          "start": 20,
          "end": 52
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
            "value": "chatsGrouped",
            "loc": {
              "start": 58,
              "end": 70
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 71,
                  "end": 76
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 79,
                    "end": 84
                  }
                },
                "loc": {
                  "start": 78,
                  "end": 84
                }
              },
              "loc": {
                "start": 71,
                "end": 84
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
                    "start": 92,
                    "end": 97
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
                          "start": 108,
                          "end": 114
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 108,
                        "end": 114
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 123,
                          "end": 127
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
                                "start": 142,
                                "end": 144
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 142,
                              "end": 144
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "chatsCount",
                              "loc": {
                                "start": 157,
                                "end": 167
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 157,
                              "end": 167
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "user",
                              "loc": {
                                "start": 180,
                                "end": 184
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
                                      "start": 203,
                                      "end": 205
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 203,
                                    "end": 205
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "bannerImage",
                                    "loc": {
                                      "start": 222,
                                      "end": 233
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 222,
                                    "end": 233
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "handle",
                                    "loc": {
                                      "start": 250,
                                      "end": 256
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 250,
                                    "end": 256
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "isBot",
                                    "loc": {
                                      "start": 273,
                                      "end": 278
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 273,
                                    "end": 278
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "name",
                                    "loc": {
                                      "start": 295,
                                      "end": 299
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 295,
                                    "end": 299
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "profileImage",
                                    "loc": {
                                      "start": 316,
                                      "end": 328
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 316,
                                    "end": 328
                                  }
                                }
                              ],
                              "loc": {
                                "start": 185,
                                "end": 342
                              }
                            },
                            "loc": {
                              "start": 180,
                              "end": 342
                            }
                          }
                        ],
                        "loc": {
                          "start": 128,
                          "end": 352
                        }
                      },
                      "loc": {
                        "start": 123,
                        "end": 352
                      }
                    }
                  ],
                  "loc": {
                    "start": 98,
                    "end": 358
                  }
                },
                "loc": {
                  "start": 92,
                  "end": 358
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 363,
                    "end": 371
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
                          "start": 382,
                          "end": 391
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 382,
                        "end": 391
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasNextPage",
                        "loc": {
                          "start": 400,
                          "end": 411
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 400,
                        "end": 411
                      }
                    }
                  ],
                  "loc": {
                    "start": 372,
                    "end": 417
                  }
                },
                "loc": {
                  "start": 363,
                  "end": 417
                }
              }
            ],
            "loc": {
              "start": 86,
              "end": 421
            }
          },
          "loc": {
            "start": 58,
            "end": 421
          }
        }
      ],
      "loc": {
        "start": 54,
        "end": 423
      }
    },
    "loc": {
      "start": 1,
      "end": 423
    }
  },
  "variableValues": {},
  "path": {
    "key": "chatsGrouped_findMany"
  }
} as const;
