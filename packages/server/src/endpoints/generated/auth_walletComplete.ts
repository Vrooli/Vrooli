export const auth_walletComplete = {
  "fieldName": "walletComplete",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "walletComplete",
        "loc": {
          "start": 420,
          "end": 434
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 435,
              "end": 440
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 443,
                "end": 448
              }
            },
            "loc": {
              "start": 442,
              "end": 448
            }
          },
          "loc": {
            "start": 435,
            "end": 448
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
              "value": "firstLogIn",
              "loc": {
                "start": 456,
                "end": 466
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 456,
              "end": 466
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "session",
              "loc": {
                "start": 471,
                "end": 478
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Session_full",
                    "loc": {
                      "start": 492,
                      "end": 504
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 489,
                    "end": 504
                  }
                }
              ],
              "loc": {
                "start": 479,
                "end": 510
              }
            },
            "loc": {
              "start": 471,
              "end": 510
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wallet",
              "loc": {
                "start": 515,
                "end": 521
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Wallet_common",
                    "loc": {
                      "start": 535,
                      "end": 548
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 532,
                    "end": 548
                  }
                }
              ],
              "loc": {
                "start": 522,
                "end": 554
              }
            },
            "loc": {
              "start": 515,
              "end": 554
            }
          }
        ],
        "loc": {
          "start": 450,
          "end": 558
        }
      },
      "loc": {
        "start": 420,
        "end": 558
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isLoggedIn",
        "loc": {
          "start": 36,
          "end": 46
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 36,
        "end": 46
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timeZone",
        "loc": {
          "start": 47,
          "end": 55
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 47,
        "end": 55
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "users",
        "loc": {
          "start": 56,
          "end": 61
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
              "value": "apisCount",
              "loc": {
                "start": 68,
                "end": 77
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 68,
              "end": 77
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "codesCount",
              "loc": {
                "start": 82,
                "end": 92
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 82,
              "end": 92
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 97,
                "end": 103
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 97,
              "end": 103
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasPremium",
              "loc": {
                "start": 108,
                "end": 118
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 108,
              "end": 118
            }
          },
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
              "value": "languages",
              "loc": {
                "start": 130,
                "end": 139
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 130,
              "end": 139
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membershipsCount",
              "loc": {
                "start": 144,
                "end": 160
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 144,
              "end": 160
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 165,
                "end": 169
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 165,
              "end": 169
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notesCount",
              "loc": {
                "start": 174,
                "end": 184
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 174,
              "end": 184
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectsCount",
              "loc": {
                "start": 189,
                "end": 202
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 189,
              "end": 202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsAskedCount",
              "loc": {
                "start": 207,
                "end": 226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 207,
              "end": 226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routinesCount",
              "loc": {
                "start": 231,
                "end": 244
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 231,
              "end": 244
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardsCount",
              "loc": {
                "start": 249,
                "end": 263
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 249,
              "end": 263
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "theme",
              "loc": {
                "start": 268,
                "end": 273
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 268,
              "end": 273
            }
          }
        ],
        "loc": {
          "start": 62,
          "end": 275
        }
      },
      "loc": {
        "start": 56,
        "end": 275
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 313,
          "end": 315
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 313,
        "end": 315
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 316,
          "end": 320
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 316,
        "end": 320
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "publicAddress",
        "loc": {
          "start": 321,
          "end": 334
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 321,
        "end": 334
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "stakingAddress",
        "loc": {
          "start": 335,
          "end": 349
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 335,
        "end": 349
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "verified",
        "loc": {
          "start": 350,
          "end": 358
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 350,
        "end": 358
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Session_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Session_full",
        "loc": {
          "start": 10,
          "end": 22
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Session",
          "loc": {
            "start": 26,
            "end": 33
          }
        },
        "loc": {
          "start": 26,
          "end": 33
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLoggedIn",
              "loc": {
                "start": 36,
                "end": 46
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 36,
              "end": 46
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeZone",
              "loc": {
                "start": 47,
                "end": 55
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 47,
              "end": 55
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 56,
                "end": 61
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
                    "value": "apisCount",
                    "loc": {
                      "start": 68,
                      "end": 77
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 68,
                    "end": 77
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "codesCount",
                    "loc": {
                      "start": 82,
                      "end": 92
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 82,
                    "end": 92
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 97,
                      "end": 103
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 97,
                    "end": 103
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasPremium",
                    "loc": {
                      "start": 108,
                      "end": 118
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 108,
                    "end": 118
                  }
                },
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
                    "value": "languages",
                    "loc": {
                      "start": 130,
                      "end": 139
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 130,
                    "end": 139
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membershipsCount",
                    "loc": {
                      "start": 144,
                      "end": 160
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 144,
                    "end": 160
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 165,
                      "end": 169
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 165,
                    "end": 169
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "notesCount",
                    "loc": {
                      "start": 174,
                      "end": 184
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 174,
                    "end": 184
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectsCount",
                    "loc": {
                      "start": 189,
                      "end": 202
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 189,
                    "end": 202
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "questionsAskedCount",
                    "loc": {
                      "start": 207,
                      "end": 226
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 207,
                    "end": 226
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routinesCount",
                    "loc": {
                      "start": 231,
                      "end": 244
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 231,
                    "end": 244
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardsCount",
                    "loc": {
                      "start": 249,
                      "end": 263
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 249,
                    "end": 263
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "theme",
                    "loc": {
                      "start": 268,
                      "end": 273
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 268,
                    "end": 273
                  }
                }
              ],
              "loc": {
                "start": 62,
                "end": 275
              }
            },
            "loc": {
              "start": 56,
              "end": 275
            }
          }
        ],
        "loc": {
          "start": 34,
          "end": 277
        }
      },
      "loc": {
        "start": 1,
        "end": 277
      }
    },
    "Wallet_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Wallet_common",
        "loc": {
          "start": 287,
          "end": 300
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Wallet",
          "loc": {
            "start": 304,
            "end": 310
          }
        },
        "loc": {
          "start": 304,
          "end": 310
        }
      },
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
                "start": 313,
                "end": 315
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 313,
              "end": 315
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 316,
                "end": 320
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 316,
              "end": 320
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "publicAddress",
              "loc": {
                "start": 321,
                "end": 334
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 321,
              "end": 334
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stakingAddress",
              "loc": {
                "start": 335,
                "end": 349
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 335,
              "end": 349
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "verified",
              "loc": {
                "start": 350,
                "end": 358
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 350,
              "end": 358
            }
          }
        ],
        "loc": {
          "start": 311,
          "end": 360
        }
      },
      "loc": {
        "start": 278,
        "end": 360
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "mutation",
    "name": {
      "kind": "Name",
      "value": "walletComplete",
      "loc": {
        "start": 371,
        "end": 385
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
              "start": 387,
              "end": 392
            }
          },
          "loc": {
            "start": 386,
            "end": 392
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "WalletCompleteInput",
              "loc": {
                "start": 394,
                "end": 413
              }
            },
            "loc": {
              "start": 394,
              "end": 413
            }
          },
          "loc": {
            "start": 394,
            "end": 414
          }
        },
        "directives": [],
        "loc": {
          "start": 386,
          "end": 414
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
            "value": "walletComplete",
            "loc": {
              "start": 420,
              "end": 434
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 435,
                  "end": 440
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 443,
                    "end": 448
                  }
                },
                "loc": {
                  "start": 442,
                  "end": 448
                }
              },
              "loc": {
                "start": 435,
                "end": 448
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
                  "value": "firstLogIn",
                  "loc": {
                    "start": 456,
                    "end": 466
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 456,
                  "end": 466
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "session",
                  "loc": {
                    "start": 471,
                    "end": 478
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Session_full",
                        "loc": {
                          "start": 492,
                          "end": 504
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 489,
                        "end": 504
                      }
                    }
                  ],
                  "loc": {
                    "start": 479,
                    "end": 510
                  }
                },
                "loc": {
                  "start": 471,
                  "end": 510
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "wallet",
                  "loc": {
                    "start": 515,
                    "end": 521
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Wallet_common",
                        "loc": {
                          "start": 535,
                          "end": 548
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 532,
                        "end": 548
                      }
                    }
                  ],
                  "loc": {
                    "start": 522,
                    "end": 554
                  }
                },
                "loc": {
                  "start": 515,
                  "end": 554
                }
              }
            ],
            "loc": {
              "start": 450,
              "end": 558
            }
          },
          "loc": {
            "start": 420,
            "end": 558
          }
        }
      ],
      "loc": {
        "start": 416,
        "end": 560
      }
    },
    "loc": {
      "start": 362,
      "end": 560
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_walletComplete"
  }
} as const;
