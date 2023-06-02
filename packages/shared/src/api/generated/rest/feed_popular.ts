export const feed_popular = {
  "fieldName": "popular",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "popular",
        "loc": {
          "start": 8102,
          "end": 8109
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8110,
              "end": 8115
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 8118,
                "end": 8123
              }
            },
            "loc": {
              "start": 8117,
              "end": 8123
            }
          },
          "loc": {
            "start": 8110,
            "end": 8123
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
              "value": "apis",
              "loc": {
                "start": 8131,
                "end": 8135
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
                    "value": "Api_list",
                    "loc": {
                      "start": 8149,
                      "end": 8157
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8146,
                    "end": 8157
                  }
                }
              ],
              "loc": {
                "start": 8136,
                "end": 8163
              }
            },
            "loc": {
              "start": 8131,
              "end": 8163
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notes",
              "loc": {
                "start": 8168,
                "end": 8173
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
                    "value": "Note_list",
                    "loc": {
                      "start": 8187,
                      "end": 8196
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8184,
                    "end": 8196
                  }
                }
              ],
              "loc": {
                "start": 8174,
                "end": 8202
              }
            },
            "loc": {
              "start": 8168,
              "end": 8202
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "organizations",
              "loc": {
                "start": 8207,
                "end": 8220
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
                    "value": "Organization_list",
                    "loc": {
                      "start": 8234,
                      "end": 8251
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8231,
                    "end": 8251
                  }
                }
              ],
              "loc": {
                "start": 8221,
                "end": 8257
              }
            },
            "loc": {
              "start": 8207,
              "end": 8257
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projects",
              "loc": {
                "start": 8262,
                "end": 8270
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
                    "value": "Project_list",
                    "loc": {
                      "start": 8284,
                      "end": 8296
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8281,
                    "end": 8296
                  }
                }
              ],
              "loc": {
                "start": 8271,
                "end": 8302
              }
            },
            "loc": {
              "start": 8262,
              "end": 8302
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questions",
              "loc": {
                "start": 8307,
                "end": 8316
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
                    "value": "Question_list",
                    "loc": {
                      "start": 8330,
                      "end": 8343
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8327,
                    "end": 8343
                  }
                }
              ],
              "loc": {
                "start": 8317,
                "end": 8349
              }
            },
            "loc": {
              "start": 8307,
              "end": 8349
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routines",
              "loc": {
                "start": 8354,
                "end": 8362
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
                    "value": "Routine_list",
                    "loc": {
                      "start": 8376,
                      "end": 8388
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8373,
                    "end": 8388
                  }
                }
              ],
              "loc": {
                "start": 8363,
                "end": 8394
              }
            },
            "loc": {
              "start": 8354,
              "end": 8394
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContracts",
              "loc": {
                "start": 8399,
                "end": 8413
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
                    "value": "SmartContract_list",
                    "loc": {
                      "start": 8427,
                      "end": 8445
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8424,
                    "end": 8445
                  }
                }
              ],
              "loc": {
                "start": 8414,
                "end": 8451
              }
            },
            "loc": {
              "start": 8399,
              "end": 8451
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standards",
              "loc": {
                "start": 8456,
                "end": 8465
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
                    "value": "Standard_list",
                    "loc": {
                      "start": 8479,
                      "end": 8492
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8476,
                    "end": 8492
                  }
                }
              ],
              "loc": {
                "start": 8466,
                "end": 8498
              }
            },
            "loc": {
              "start": 8456,
              "end": 8498
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 8503,
                "end": 8508
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
                    "value": "User_list",
                    "loc": {
                      "start": 8522,
                      "end": 8531
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8519,
                    "end": 8531
                  }
                }
              ],
              "loc": {
                "start": 8509,
                "end": 8537
              }
            },
            "loc": {
              "start": 8503,
              "end": 8537
            }
          }
        ],
        "loc": {
          "start": 8125,
          "end": 8541
        }
      },
      "loc": {
        "start": 8102,
        "end": 8541
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Api_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_list",
        "loc": {
          "start": 9,
          "end": 17
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 21,
            "end": 24
          }
        },
        "loc": {
          "start": 21,
          "end": 24
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
              "value": "versions",
              "loc": {
                "start": 27,
                "end": 35
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
                    "value": "translations",
                    "loc": {
                      "start": 42,
                      "end": 54
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
                            "start": 65,
                            "end": 67
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 65,
                          "end": 67
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 76,
                            "end": 84
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 76,
                          "end": 84
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "details",
                          "loc": {
                            "start": 93,
                            "end": 100
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 93,
                          "end": 100
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 109,
                            "end": 113
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 109,
                          "end": 113
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "summary",
                          "loc": {
                            "start": 122,
                            "end": 129
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 122,
                          "end": 129
                        }
                      }
                    ],
                    "loc": {
                      "start": 55,
                      "end": 135
                    }
                  },
                  "loc": {
                    "start": 42,
                    "end": 135
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 140,
                      "end": 142
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 140,
                    "end": 142
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 147,
                      "end": 157
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 147,
                    "end": 157
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 162,
                      "end": 172
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 162,
                    "end": 172
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "callLink",
                    "loc": {
                      "start": 177,
                      "end": 185
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 177,
                    "end": 185
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 190,
                      "end": 203
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 190,
                    "end": 203
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "documentationLink",
                    "loc": {
                      "start": 208,
                      "end": 225
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 208,
                    "end": 225
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 230,
                      "end": 240
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 230,
                    "end": 240
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 245,
                      "end": 253
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 245,
                    "end": 253
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 258,
                      "end": 267
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 258,
                    "end": 267
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 272,
                      "end": 284
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 272,
                    "end": 284
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 289,
                      "end": 301
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 289,
                    "end": 301
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 306,
                      "end": 318
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 306,
                    "end": 318
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 323,
                      "end": 326
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
                          "value": "canComment",
                          "loc": {
                            "start": 337,
                            "end": 347
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 337,
                          "end": 347
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 356,
                            "end": 363
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 356,
                          "end": 363
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 372,
                            "end": 381
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 372,
                          "end": 381
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 390,
                            "end": 399
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 390,
                          "end": 399
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 408,
                            "end": 417
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 408,
                          "end": 417
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 426,
                            "end": 432
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 426,
                          "end": 432
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 441,
                            "end": 448
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 441,
                          "end": 448
                        }
                      }
                    ],
                    "loc": {
                      "start": 327,
                      "end": 454
                    }
                  },
                  "loc": {
                    "start": 323,
                    "end": 454
                  }
                }
              ],
              "loc": {
                "start": 36,
                "end": 456
              }
            },
            "loc": {
              "start": 27,
              "end": 456
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 457,
                "end": 459
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 457,
              "end": 459
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 460,
                "end": 470
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 460,
              "end": 470
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 471,
                "end": 481
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 471,
              "end": 481
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 482,
                "end": 491
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 482,
              "end": 491
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 492,
                "end": 503
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 492,
              "end": 503
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 504,
                "end": 510
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
                    "value": "Label_list",
                    "loc": {
                      "start": 520,
                      "end": 530
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 517,
                    "end": 530
                  }
                }
              ],
              "loc": {
                "start": 511,
                "end": 532
              }
            },
            "loc": {
              "start": 504,
              "end": 532
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 533,
                "end": 538
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 552,
                        "end": 564
                      }
                    },
                    "loc": {
                      "start": 552,
                      "end": 564
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 578,
                            "end": 594
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 575,
                          "end": 594
                        }
                      }
                    ],
                    "loc": {
                      "start": 565,
                      "end": 600
                    }
                  },
                  "loc": {
                    "start": 545,
                    "end": 600
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 612,
                        "end": 616
                      }
                    },
                    "loc": {
                      "start": 612,
                      "end": 616
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 630,
                            "end": 638
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 627,
                          "end": 638
                        }
                      }
                    ],
                    "loc": {
                      "start": 617,
                      "end": 644
                    }
                  },
                  "loc": {
                    "start": 605,
                    "end": 644
                  }
                }
              ],
              "loc": {
                "start": 539,
                "end": 646
              }
            },
            "loc": {
              "start": 533,
              "end": 646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 647,
                "end": 658
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 647,
              "end": 658
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 659,
                "end": 673
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 659,
              "end": 673
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 674,
                "end": 679
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 674,
              "end": 679
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 680,
                "end": 689
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 680,
              "end": 689
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 690,
                "end": 694
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 704,
                      "end": 712
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 701,
                    "end": 712
                  }
                }
              ],
              "loc": {
                "start": 695,
                "end": 714
              }
            },
            "loc": {
              "start": 690,
              "end": 714
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 715,
                "end": 729
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 715,
              "end": 729
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 730,
                "end": 735
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 730,
              "end": 735
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 736,
                "end": 739
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
                    "value": "canDelete",
                    "loc": {
                      "start": 746,
                      "end": 755
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 746,
                    "end": 755
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 760,
                      "end": 771
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 760,
                    "end": 771
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 776,
                      "end": 787
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 776,
                    "end": 787
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 792,
                      "end": 801
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 792,
                    "end": 801
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 806,
                      "end": 813
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 806,
                    "end": 813
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 818,
                      "end": 826
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 818,
                    "end": 826
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 831,
                      "end": 843
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 831,
                    "end": 843
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 848,
                      "end": 856
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 848,
                    "end": 856
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 861,
                      "end": 869
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 861,
                    "end": 869
                  }
                }
              ],
              "loc": {
                "start": 740,
                "end": 871
              }
            },
            "loc": {
              "start": 736,
              "end": 871
            }
          }
        ],
        "loc": {
          "start": 25,
          "end": 873
        }
      },
      "loc": {
        "start": 0,
        "end": 873
      }
    },
    "Api_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_nav",
        "loc": {
          "start": 883,
          "end": 890
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 894,
            "end": 897
          }
        },
        "loc": {
          "start": 894,
          "end": 897
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
                "start": 900,
                "end": 902
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 900,
              "end": 902
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 903,
                "end": 912
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 903,
              "end": 912
            }
          }
        ],
        "loc": {
          "start": 898,
          "end": 914
        }
      },
      "loc": {
        "start": 874,
        "end": 914
      }
    },
    "Label_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_full",
        "loc": {
          "start": 924,
          "end": 934
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 938,
            "end": 943
          }
        },
        "loc": {
          "start": 938,
          "end": 943
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
              "value": "apisCount",
              "loc": {
                "start": 946,
                "end": 955
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 946,
              "end": 955
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModesCount",
              "loc": {
                "start": 956,
                "end": 971
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 956,
              "end": 971
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 972,
                "end": 983
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 972,
              "end": 983
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "meetingsCount",
              "loc": {
                "start": 984,
                "end": 997
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 984,
              "end": 997
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notesCount",
              "loc": {
                "start": 998,
                "end": 1008
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 998,
              "end": 1008
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectsCount",
              "loc": {
                "start": 1009,
                "end": 1022
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1009,
              "end": 1022
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routinesCount",
              "loc": {
                "start": 1023,
                "end": 1036
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1023,
              "end": 1036
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "schedulesCount",
              "loc": {
                "start": 1037,
                "end": 1051
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1037,
              "end": 1051
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractsCount",
              "loc": {
                "start": 1052,
                "end": 1071
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1052,
              "end": 1071
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardsCount",
              "loc": {
                "start": 1072,
                "end": 1086
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1072,
              "end": 1086
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1087,
                "end": 1089
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1087,
              "end": 1089
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1090,
                "end": 1100
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1090,
              "end": 1100
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1101,
                "end": 1111
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1101,
              "end": 1111
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 1112,
                "end": 1117
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1112,
              "end": 1117
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 1118,
                "end": 1123
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1118,
              "end": 1123
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1124,
                "end": 1129
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1143,
                        "end": 1155
                      }
                    },
                    "loc": {
                      "start": 1143,
                      "end": 1155
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1169,
                            "end": 1185
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1166,
                          "end": 1185
                        }
                      }
                    ],
                    "loc": {
                      "start": 1156,
                      "end": 1191
                    }
                  },
                  "loc": {
                    "start": 1136,
                    "end": 1191
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1203,
                        "end": 1207
                      }
                    },
                    "loc": {
                      "start": 1203,
                      "end": 1207
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1221,
                            "end": 1229
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1218,
                          "end": 1229
                        }
                      }
                    ],
                    "loc": {
                      "start": 1208,
                      "end": 1235
                    }
                  },
                  "loc": {
                    "start": 1196,
                    "end": 1235
                  }
                }
              ],
              "loc": {
                "start": 1130,
                "end": 1237
              }
            },
            "loc": {
              "start": 1124,
              "end": 1237
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1238,
                "end": 1241
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
                    "value": "canDelete",
                    "loc": {
                      "start": 1248,
                      "end": 1257
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1248,
                    "end": 1257
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1262,
                      "end": 1271
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1262,
                    "end": 1271
                  }
                }
              ],
              "loc": {
                "start": 1242,
                "end": 1273
              }
            },
            "loc": {
              "start": 1238,
              "end": 1273
            }
          }
        ],
        "loc": {
          "start": 944,
          "end": 1275
        }
      },
      "loc": {
        "start": 915,
        "end": 1275
      }
    },
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 1285,
          "end": 1295
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 1299,
            "end": 1304
          }
        },
        "loc": {
          "start": 1299,
          "end": 1304
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
                "start": 1307,
                "end": 1309
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1307,
              "end": 1309
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1310,
                "end": 1320
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1310,
              "end": 1320
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1321,
                "end": 1331
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1321,
              "end": 1331
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 1332,
                "end": 1337
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1332,
              "end": 1337
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 1338,
                "end": 1343
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1338,
              "end": 1343
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1344,
                "end": 1349
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1363,
                        "end": 1375
                      }
                    },
                    "loc": {
                      "start": 1363,
                      "end": 1375
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 1389,
                            "end": 1405
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1386,
                          "end": 1405
                        }
                      }
                    ],
                    "loc": {
                      "start": 1376,
                      "end": 1411
                    }
                  },
                  "loc": {
                    "start": 1356,
                    "end": 1411
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 1423,
                        "end": 1427
                      }
                    },
                    "loc": {
                      "start": 1423,
                      "end": 1427
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 1441,
                            "end": 1449
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1438,
                          "end": 1449
                        }
                      }
                    ],
                    "loc": {
                      "start": 1428,
                      "end": 1455
                    }
                  },
                  "loc": {
                    "start": 1416,
                    "end": 1455
                  }
                }
              ],
              "loc": {
                "start": 1350,
                "end": 1457
              }
            },
            "loc": {
              "start": 1344,
              "end": 1457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1458,
                "end": 1461
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
                    "value": "canDelete",
                    "loc": {
                      "start": 1468,
                      "end": 1477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1468,
                    "end": 1477
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1482,
                      "end": 1491
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1482,
                    "end": 1491
                  }
                }
              ],
              "loc": {
                "start": 1462,
                "end": 1493
              }
            },
            "loc": {
              "start": 1458,
              "end": 1493
            }
          }
        ],
        "loc": {
          "start": 1305,
          "end": 1495
        }
      },
      "loc": {
        "start": 1276,
        "end": 1495
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 1505,
          "end": 1514
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 1518,
            "end": 1522
          }
        },
        "loc": {
          "start": 1518,
          "end": 1522
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
              "value": "versions",
              "loc": {
                "start": 1525,
                "end": 1533
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
                    "value": "translations",
                    "loc": {
                      "start": 1540,
                      "end": 1552
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
                            "start": 1563,
                            "end": 1565
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1563,
                          "end": 1565
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1574,
                            "end": 1582
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1574,
                          "end": 1582
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1591,
                            "end": 1602
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1591,
                          "end": 1602
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1611,
                            "end": 1615
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1611,
                          "end": 1615
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 1624,
                            "end": 1628
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1624,
                          "end": 1628
                        }
                      }
                    ],
                    "loc": {
                      "start": 1553,
                      "end": 1634
                    }
                  },
                  "loc": {
                    "start": 1540,
                    "end": 1634
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1639,
                      "end": 1641
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1639,
                    "end": 1641
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1646,
                      "end": 1656
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1646,
                    "end": 1656
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1661,
                      "end": 1671
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1661,
                    "end": 1671
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1676,
                      "end": 1684
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1676,
                    "end": 1684
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1689,
                      "end": 1698
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1689,
                    "end": 1698
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1703,
                      "end": 1715
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1703,
                    "end": 1715
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1720,
                      "end": 1732
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1720,
                    "end": 1732
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1737,
                      "end": 1749
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1737,
                    "end": 1749
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 1754,
                      "end": 1757
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
                          "value": "canComment",
                          "loc": {
                            "start": 1768,
                            "end": 1778
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1768,
                          "end": 1778
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 1787,
                            "end": 1794
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1787,
                          "end": 1794
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 1803,
                            "end": 1812
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1803,
                          "end": 1812
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 1821,
                            "end": 1830
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1821,
                          "end": 1830
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 1839,
                            "end": 1848
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1839,
                          "end": 1848
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 1857,
                            "end": 1863
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1857,
                          "end": 1863
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 1872,
                            "end": 1879
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1872,
                          "end": 1879
                        }
                      }
                    ],
                    "loc": {
                      "start": 1758,
                      "end": 1885
                    }
                  },
                  "loc": {
                    "start": 1754,
                    "end": 1885
                  }
                }
              ],
              "loc": {
                "start": 1534,
                "end": 1887
              }
            },
            "loc": {
              "start": 1525,
              "end": 1887
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1888,
                "end": 1890
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1888,
              "end": 1890
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1891,
                "end": 1901
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1891,
              "end": 1901
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1902,
                "end": 1912
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1902,
              "end": 1912
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1913,
                "end": 1922
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1913,
              "end": 1922
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1923,
                "end": 1934
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1923,
              "end": 1934
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1935,
                "end": 1941
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
                    "value": "Label_list",
                    "loc": {
                      "start": 1951,
                      "end": 1961
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1948,
                    "end": 1961
                  }
                }
              ],
              "loc": {
                "start": 1942,
                "end": 1963
              }
            },
            "loc": {
              "start": 1935,
              "end": 1963
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1964,
                "end": 1969
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 1983,
                        "end": 1995
                      }
                    },
                    "loc": {
                      "start": 1983,
                      "end": 1995
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 2009,
                            "end": 2025
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2006,
                          "end": 2025
                        }
                      }
                    ],
                    "loc": {
                      "start": 1996,
                      "end": 2031
                    }
                  },
                  "loc": {
                    "start": 1976,
                    "end": 2031
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 2043,
                        "end": 2047
                      }
                    },
                    "loc": {
                      "start": 2043,
                      "end": 2047
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 2061,
                            "end": 2069
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 2058,
                          "end": 2069
                        }
                      }
                    ],
                    "loc": {
                      "start": 2048,
                      "end": 2075
                    }
                  },
                  "loc": {
                    "start": 2036,
                    "end": 2075
                  }
                }
              ],
              "loc": {
                "start": 1970,
                "end": 2077
              }
            },
            "loc": {
              "start": 1964,
              "end": 2077
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 2078,
                "end": 2089
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2078,
              "end": 2089
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 2090,
                "end": 2104
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2090,
              "end": 2104
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 2105,
                "end": 2110
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2105,
              "end": 2110
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2111,
                "end": 2120
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2111,
              "end": 2120
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2121,
                "end": 2125
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 2135,
                      "end": 2143
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2132,
                    "end": 2143
                  }
                }
              ],
              "loc": {
                "start": 2126,
                "end": 2145
              }
            },
            "loc": {
              "start": 2121,
              "end": 2145
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 2146,
                "end": 2160
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2146,
              "end": 2160
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 2161,
                "end": 2166
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2161,
              "end": 2166
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2167,
                "end": 2170
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
                    "value": "canDelete",
                    "loc": {
                      "start": 2177,
                      "end": 2186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2177,
                    "end": 2186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2191,
                      "end": 2202
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2191,
                    "end": 2202
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 2207,
                      "end": 2218
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2207,
                    "end": 2218
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2223,
                      "end": 2232
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2223,
                    "end": 2232
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2237,
                      "end": 2244
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2237,
                    "end": 2244
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 2249,
                      "end": 2257
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2249,
                    "end": 2257
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2262,
                      "end": 2274
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2262,
                    "end": 2274
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2279,
                      "end": 2287
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2279,
                    "end": 2287
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 2292,
                      "end": 2300
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2292,
                    "end": 2300
                  }
                }
              ],
              "loc": {
                "start": 2171,
                "end": 2302
              }
            },
            "loc": {
              "start": 2167,
              "end": 2302
            }
          }
        ],
        "loc": {
          "start": 1523,
          "end": 2304
        }
      },
      "loc": {
        "start": 1496,
        "end": 2304
      }
    },
    "Note_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_nav",
        "loc": {
          "start": 2314,
          "end": 2322
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 2326,
            "end": 2330
          }
        },
        "loc": {
          "start": 2326,
          "end": 2330
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
                "start": 2333,
                "end": 2335
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2333,
              "end": 2335
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2336,
                "end": 2345
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2336,
              "end": 2345
            }
          }
        ],
        "loc": {
          "start": 2331,
          "end": 2347
        }
      },
      "loc": {
        "start": 2305,
        "end": 2347
      }
    },
    "Organization_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_list",
        "loc": {
          "start": 2357,
          "end": 2374
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 2378,
            "end": 2390
          }
        },
        "loc": {
          "start": 2378,
          "end": 2390
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
                "start": 2393,
                "end": 2395
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2393,
              "end": 2395
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2396,
                "end": 2402
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2396,
              "end": 2402
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2403,
                "end": 2413
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2403,
              "end": 2413
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2414,
                "end": 2424
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2414,
              "end": 2424
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 2425,
                "end": 2443
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2425,
              "end": 2443
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2444,
                "end": 2453
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2444,
              "end": 2453
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 2454,
                "end": 2467
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2454,
              "end": 2467
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 2468,
                "end": 2480
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2468,
              "end": 2480
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 2481,
                "end": 2493
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2481,
              "end": 2493
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2494,
                "end": 2503
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2494,
              "end": 2503
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2504,
                "end": 2508
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 2518,
                      "end": 2526
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2515,
                    "end": 2526
                  }
                }
              ],
              "loc": {
                "start": 2509,
                "end": 2528
              }
            },
            "loc": {
              "start": 2504,
              "end": 2528
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2529,
                "end": 2541
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
                      "start": 2548,
                      "end": 2550
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2548,
                    "end": 2550
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2555,
                      "end": 2563
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2555,
                    "end": 2563
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 2568,
                      "end": 2571
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2568,
                    "end": 2571
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2576,
                      "end": 2580
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2576,
                    "end": 2580
                  }
                }
              ],
              "loc": {
                "start": 2542,
                "end": 2582
              }
            },
            "loc": {
              "start": 2529,
              "end": 2582
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2583,
                "end": 2586
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
                    "value": "canAddMembers",
                    "loc": {
                      "start": 2593,
                      "end": 2606
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2593,
                    "end": 2606
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2611,
                      "end": 2620
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2611,
                    "end": 2620
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2625,
                      "end": 2636
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2625,
                    "end": 2636
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2641,
                      "end": 2650
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2641,
                    "end": 2650
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2655,
                      "end": 2664
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2655,
                    "end": 2664
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2669,
                      "end": 2676
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2669,
                    "end": 2676
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2681,
                      "end": 2693
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2681,
                    "end": 2693
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2698,
                      "end": 2706
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2698,
                    "end": 2706
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 2711,
                      "end": 2725
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
                            "start": 2736,
                            "end": 2738
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2736,
                          "end": 2738
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2747,
                            "end": 2757
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2747,
                          "end": 2757
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2766,
                            "end": 2776
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2766,
                          "end": 2776
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 2785,
                            "end": 2792
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2785,
                          "end": 2792
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 2801,
                            "end": 2812
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2801,
                          "end": 2812
                        }
                      }
                    ],
                    "loc": {
                      "start": 2726,
                      "end": 2818
                    }
                  },
                  "loc": {
                    "start": 2711,
                    "end": 2818
                  }
                }
              ],
              "loc": {
                "start": 2587,
                "end": 2820
              }
            },
            "loc": {
              "start": 2583,
              "end": 2820
            }
          }
        ],
        "loc": {
          "start": 2391,
          "end": 2822
        }
      },
      "loc": {
        "start": 2348,
        "end": 2822
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 2832,
          "end": 2848
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 2852,
            "end": 2864
          }
        },
        "loc": {
          "start": 2852,
          "end": 2864
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
                "start": 2867,
                "end": 2869
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2867,
              "end": 2869
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2870,
                "end": 2876
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2870,
              "end": 2876
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2877,
                "end": 2880
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
                    "value": "canAddMembers",
                    "loc": {
                      "start": 2887,
                      "end": 2900
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2887,
                    "end": 2900
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2905,
                      "end": 2914
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2905,
                    "end": 2914
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2919,
                      "end": 2930
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2919,
                    "end": 2930
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2935,
                      "end": 2944
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2935,
                    "end": 2944
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2949,
                      "end": 2958
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2949,
                    "end": 2958
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2963,
                      "end": 2970
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2963,
                    "end": 2970
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2975,
                      "end": 2987
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2975,
                    "end": 2987
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2992,
                      "end": 3000
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2992,
                    "end": 3000
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 3005,
                      "end": 3019
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
                            "start": 3030,
                            "end": 3032
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3030,
                          "end": 3032
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3041,
                            "end": 3051
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3041,
                          "end": 3051
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3060,
                            "end": 3070
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3060,
                          "end": 3070
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 3079,
                            "end": 3086
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3079,
                          "end": 3086
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 3095,
                            "end": 3106
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3095,
                          "end": 3106
                        }
                      }
                    ],
                    "loc": {
                      "start": 3020,
                      "end": 3112
                    }
                  },
                  "loc": {
                    "start": 3005,
                    "end": 3112
                  }
                }
              ],
              "loc": {
                "start": 2881,
                "end": 3114
              }
            },
            "loc": {
              "start": 2877,
              "end": 3114
            }
          }
        ],
        "loc": {
          "start": 2865,
          "end": 3116
        }
      },
      "loc": {
        "start": 2823,
        "end": 3116
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 3126,
          "end": 3138
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3142,
            "end": 3149
          }
        },
        "loc": {
          "start": 3142,
          "end": 3149
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
              "value": "versions",
              "loc": {
                "start": 3152,
                "end": 3160
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
                    "value": "translations",
                    "loc": {
                      "start": 3167,
                      "end": 3179
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
                            "start": 3190,
                            "end": 3192
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3190,
                          "end": 3192
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 3201,
                            "end": 3209
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3201,
                          "end": 3209
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 3218,
                            "end": 3229
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3218,
                          "end": 3229
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 3238,
                            "end": 3242
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3238,
                          "end": 3242
                        }
                      }
                    ],
                    "loc": {
                      "start": 3180,
                      "end": 3248
                    }
                  },
                  "loc": {
                    "start": 3167,
                    "end": 3248
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 3253,
                      "end": 3255
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3253,
                    "end": 3255
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 3260,
                      "end": 3270
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3260,
                    "end": 3270
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 3275,
                      "end": 3285
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3275,
                    "end": 3285
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 3290,
                      "end": 3306
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3290,
                    "end": 3306
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 3311,
                      "end": 3319
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3311,
                    "end": 3319
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 3324,
                      "end": 3333
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3324,
                    "end": 3333
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 3338,
                      "end": 3350
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3338,
                    "end": 3350
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 3355,
                      "end": 3371
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3355,
                    "end": 3371
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 3376,
                      "end": 3386
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3376,
                    "end": 3386
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 3391,
                      "end": 3403
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3391,
                    "end": 3403
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 3408,
                      "end": 3420
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3408,
                    "end": 3420
                  }
                }
              ],
              "loc": {
                "start": 3161,
                "end": 3422
              }
            },
            "loc": {
              "start": 3152,
              "end": 3422
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3423,
                "end": 3425
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3423,
              "end": 3425
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3426,
                "end": 3436
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3426,
              "end": 3436
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3437,
                "end": 3447
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3437,
              "end": 3447
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3448,
                "end": 3457
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3448,
              "end": 3457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 3458,
                "end": 3469
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3458,
              "end": 3469
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 3470,
                "end": 3476
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
                    "value": "Label_list",
                    "loc": {
                      "start": 3486,
                      "end": 3496
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3483,
                    "end": 3496
                  }
                }
              ],
              "loc": {
                "start": 3477,
                "end": 3498
              }
            },
            "loc": {
              "start": 3470,
              "end": 3498
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 3499,
                "end": 3504
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 3518,
                        "end": 3530
                      }
                    },
                    "loc": {
                      "start": 3518,
                      "end": 3530
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 3544,
                            "end": 3560
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3541,
                          "end": 3560
                        }
                      }
                    ],
                    "loc": {
                      "start": 3531,
                      "end": 3566
                    }
                  },
                  "loc": {
                    "start": 3511,
                    "end": 3566
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 3578,
                        "end": 3582
                      }
                    },
                    "loc": {
                      "start": 3578,
                      "end": 3582
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 3596,
                            "end": 3604
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3593,
                          "end": 3604
                        }
                      }
                    ],
                    "loc": {
                      "start": 3583,
                      "end": 3610
                    }
                  },
                  "loc": {
                    "start": 3571,
                    "end": 3610
                  }
                }
              ],
              "loc": {
                "start": 3505,
                "end": 3612
              }
            },
            "loc": {
              "start": 3499,
              "end": 3612
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 3613,
                "end": 3624
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3613,
              "end": 3624
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 3625,
                "end": 3639
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3625,
              "end": 3639
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3640,
                "end": 3645
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3640,
              "end": 3645
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3646,
                "end": 3655
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3646,
              "end": 3655
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 3656,
                "end": 3660
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 3670,
                      "end": 3678
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3667,
                    "end": 3678
                  }
                }
              ],
              "loc": {
                "start": 3661,
                "end": 3680
              }
            },
            "loc": {
              "start": 3656,
              "end": 3680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 3681,
                "end": 3695
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3681,
              "end": 3695
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 3696,
                "end": 3701
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3696,
              "end": 3701
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 3702,
                "end": 3705
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
                    "value": "canDelete",
                    "loc": {
                      "start": 3712,
                      "end": 3721
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3712,
                    "end": 3721
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 3726,
                      "end": 3737
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3726,
                    "end": 3737
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 3742,
                      "end": 3753
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3742,
                    "end": 3753
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 3758,
                      "end": 3767
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3758,
                    "end": 3767
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 3772,
                      "end": 3779
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3772,
                    "end": 3779
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 3784,
                      "end": 3792
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3784,
                    "end": 3792
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 3797,
                      "end": 3809
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3797,
                    "end": 3809
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 3814,
                      "end": 3822
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3814,
                    "end": 3822
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 3827,
                      "end": 3835
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3827,
                    "end": 3835
                  }
                }
              ],
              "loc": {
                "start": 3706,
                "end": 3837
              }
            },
            "loc": {
              "start": 3702,
              "end": 3837
            }
          }
        ],
        "loc": {
          "start": 3150,
          "end": 3839
        }
      },
      "loc": {
        "start": 3117,
        "end": 3839
      }
    },
    "Project_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_nav",
        "loc": {
          "start": 3849,
          "end": 3860
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3864,
            "end": 3871
          }
        },
        "loc": {
          "start": 3864,
          "end": 3871
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
                "start": 3874,
                "end": 3876
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3874,
              "end": 3876
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3877,
                "end": 3886
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3877,
              "end": 3886
            }
          }
        ],
        "loc": {
          "start": 3872,
          "end": 3888
        }
      },
      "loc": {
        "start": 3840,
        "end": 3888
      }
    },
    "Question_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Question_list",
        "loc": {
          "start": 3898,
          "end": 3911
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Question",
          "loc": {
            "start": 3915,
            "end": 3923
          }
        },
        "loc": {
          "start": 3915,
          "end": 3923
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
              "value": "translations",
              "loc": {
                "start": 3926,
                "end": 3938
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
                      "start": 3945,
                      "end": 3947
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3945,
                    "end": 3947
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3952,
                      "end": 3960
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3952,
                    "end": 3960
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3965,
                      "end": 3976
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3965,
                    "end": 3976
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3981,
                      "end": 3985
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3981,
                    "end": 3985
                  }
                }
              ],
              "loc": {
                "start": 3939,
                "end": 3987
              }
            },
            "loc": {
              "start": 3926,
              "end": 3987
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3988,
                "end": 3990
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3988,
              "end": 3990
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3991,
                "end": 4001
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3991,
              "end": 4001
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4002,
                "end": 4012
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4002,
              "end": 4012
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "createdBy",
              "loc": {
                "start": 4013,
                "end": 4022
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
                      "start": 4029,
                      "end": 4031
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4029,
                    "end": 4031
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBot",
                    "loc": {
                      "start": 4036,
                      "end": 4041
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4036,
                    "end": 4041
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4046,
                      "end": 4050
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4046,
                    "end": 4050
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 4055,
                      "end": 4061
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4055,
                    "end": 4061
                  }
                }
              ],
              "loc": {
                "start": 4023,
                "end": 4063
              }
            },
            "loc": {
              "start": 4013,
              "end": 4063
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasAcceptedAnswer",
              "loc": {
                "start": 4064,
                "end": 4081
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4064,
              "end": 4081
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4082,
                "end": 4091
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4082,
              "end": 4091
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 4092,
                "end": 4097
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4092,
              "end": 4097
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 4098,
                "end": 4107
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4098,
              "end": 4107
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "answersCount",
              "loc": {
                "start": 4108,
                "end": 4120
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4108,
              "end": 4120
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4121,
                "end": 4134
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4121,
              "end": 4134
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 4135,
                "end": 4147
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4135,
              "end": 4147
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forObject",
              "loc": {
                "start": 4148,
                "end": 4157
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Api",
                      "loc": {
                        "start": 4171,
                        "end": 4174
                      }
                    },
                    "loc": {
                      "start": 4171,
                      "end": 4174
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Api_nav",
                          "loc": {
                            "start": 4188,
                            "end": 4195
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4185,
                          "end": 4195
                        }
                      }
                    ],
                    "loc": {
                      "start": 4175,
                      "end": 4201
                    }
                  },
                  "loc": {
                    "start": 4164,
                    "end": 4201
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Note",
                      "loc": {
                        "start": 4213,
                        "end": 4217
                      }
                    },
                    "loc": {
                      "start": 4213,
                      "end": 4217
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Note_nav",
                          "loc": {
                            "start": 4231,
                            "end": 4239
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4228,
                          "end": 4239
                        }
                      }
                    ],
                    "loc": {
                      "start": 4218,
                      "end": 4245
                    }
                  },
                  "loc": {
                    "start": 4206,
                    "end": 4245
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 4257,
                        "end": 4269
                      }
                    },
                    "loc": {
                      "start": 4257,
                      "end": 4269
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 4283,
                            "end": 4299
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4280,
                          "end": 4299
                        }
                      }
                    ],
                    "loc": {
                      "start": 4270,
                      "end": 4305
                    }
                  },
                  "loc": {
                    "start": 4250,
                    "end": 4305
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Project",
                      "loc": {
                        "start": 4317,
                        "end": 4324
                      }
                    },
                    "loc": {
                      "start": 4317,
                      "end": 4324
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Project_nav",
                          "loc": {
                            "start": 4338,
                            "end": 4349
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4335,
                          "end": 4349
                        }
                      }
                    ],
                    "loc": {
                      "start": 4325,
                      "end": 4355
                    }
                  },
                  "loc": {
                    "start": 4310,
                    "end": 4355
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Routine",
                      "loc": {
                        "start": 4367,
                        "end": 4374
                      }
                    },
                    "loc": {
                      "start": 4367,
                      "end": 4374
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Routine_nav",
                          "loc": {
                            "start": 4388,
                            "end": 4399
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4385,
                          "end": 4399
                        }
                      }
                    ],
                    "loc": {
                      "start": 4375,
                      "end": 4405
                    }
                  },
                  "loc": {
                    "start": 4360,
                    "end": 4405
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "SmartContract",
                      "loc": {
                        "start": 4417,
                        "end": 4430
                      }
                    },
                    "loc": {
                      "start": 4417,
                      "end": 4430
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "SmartContract_nav",
                          "loc": {
                            "start": 4444,
                            "end": 4461
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4441,
                          "end": 4461
                        }
                      }
                    ],
                    "loc": {
                      "start": 4431,
                      "end": 4467
                    }
                  },
                  "loc": {
                    "start": 4410,
                    "end": 4467
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Standard",
                      "loc": {
                        "start": 4479,
                        "end": 4487
                      }
                    },
                    "loc": {
                      "start": 4479,
                      "end": 4487
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Standard_nav",
                          "loc": {
                            "start": 4501,
                            "end": 4513
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4498,
                          "end": 4513
                        }
                      }
                    ],
                    "loc": {
                      "start": 4488,
                      "end": 4519
                    }
                  },
                  "loc": {
                    "start": 4472,
                    "end": 4519
                  }
                }
              ],
              "loc": {
                "start": 4158,
                "end": 4521
              }
            },
            "loc": {
              "start": 4148,
              "end": 4521
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 4522,
                "end": 4526
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 4536,
                      "end": 4544
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4533,
                    "end": 4544
                  }
                }
              ],
              "loc": {
                "start": 4527,
                "end": 4546
              }
            },
            "loc": {
              "start": 4522,
              "end": 4546
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4547,
                "end": 4550
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
                    "value": "reaction",
                    "loc": {
                      "start": 4557,
                      "end": 4565
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4557,
                    "end": 4565
                  }
                }
              ],
              "loc": {
                "start": 4551,
                "end": 4567
              }
            },
            "loc": {
              "start": 4547,
              "end": 4567
            }
          }
        ],
        "loc": {
          "start": 3924,
          "end": 4569
        }
      },
      "loc": {
        "start": 3889,
        "end": 4569
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 4579,
          "end": 4591
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 4595,
            "end": 4602
          }
        },
        "loc": {
          "start": 4595,
          "end": 4602
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
              "value": "versions",
              "loc": {
                "start": 4605,
                "end": 4613
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
                    "value": "translations",
                    "loc": {
                      "start": 4620,
                      "end": 4632
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
                            "start": 4643,
                            "end": 4645
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4643,
                          "end": 4645
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 4654,
                            "end": 4662
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4654,
                          "end": 4662
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4671,
                            "end": 4682
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4671,
                          "end": 4682
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 4691,
                            "end": 4703
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4691,
                          "end": 4703
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4712,
                            "end": 4716
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4712,
                          "end": 4716
                        }
                      }
                    ],
                    "loc": {
                      "start": 4633,
                      "end": 4722
                    }
                  },
                  "loc": {
                    "start": 4620,
                    "end": 4722
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4727,
                      "end": 4729
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4727,
                    "end": 4729
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4734,
                      "end": 4744
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4734,
                    "end": 4744
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4749,
                      "end": 4759
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4749,
                    "end": 4759
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 4764,
                      "end": 4775
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4764,
                    "end": 4775
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 4780,
                      "end": 4793
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4780,
                    "end": 4793
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 4798,
                      "end": 4808
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4798,
                    "end": 4808
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 4813,
                      "end": 4822
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4813,
                    "end": 4822
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 4827,
                      "end": 4835
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4827,
                    "end": 4835
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 4840,
                      "end": 4849
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4840,
                    "end": 4849
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 4854,
                      "end": 4864
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4854,
                    "end": 4864
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 4869,
                      "end": 4881
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4869,
                    "end": 4881
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 4886,
                      "end": 4900
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4886,
                    "end": 4900
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractCallData",
                    "loc": {
                      "start": 4905,
                      "end": 4926
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4905,
                    "end": 4926
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apiCallData",
                    "loc": {
                      "start": 4931,
                      "end": 4942
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4931,
                    "end": 4942
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 4947,
                      "end": 4959
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4947,
                    "end": 4959
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 4964,
                      "end": 4976
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4964,
                    "end": 4976
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 4981,
                      "end": 4994
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4981,
                    "end": 4994
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 4999,
                      "end": 5021
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4999,
                    "end": 5021
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 5026,
                      "end": 5036
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5026,
                    "end": 5036
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "inputsCount",
                    "loc": {
                      "start": 5041,
                      "end": 5052
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5041,
                    "end": 5052
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodesCount",
                    "loc": {
                      "start": 5057,
                      "end": 5067
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5057,
                    "end": 5067
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 5072,
                      "end": 5086
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5072,
                    "end": 5086
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 5091,
                      "end": 5103
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5091,
                    "end": 5103
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 5108,
                      "end": 5120
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5108,
                    "end": 5120
                  }
                }
              ],
              "loc": {
                "start": 4614,
                "end": 5122
              }
            },
            "loc": {
              "start": 4605,
              "end": 5122
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5123,
                "end": 5125
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5123,
              "end": 5125
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5126,
                "end": 5136
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5126,
              "end": 5136
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5137,
                "end": 5147
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5137,
              "end": 5147
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5148,
                "end": 5158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5148,
              "end": 5158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5159,
                "end": 5168
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5159,
              "end": 5168
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 5169,
                "end": 5180
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5169,
              "end": 5180
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 5181,
                "end": 5187
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
                    "value": "Label_list",
                    "loc": {
                      "start": 5197,
                      "end": 5207
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5194,
                    "end": 5207
                  }
                }
              ],
              "loc": {
                "start": 5188,
                "end": 5209
              }
            },
            "loc": {
              "start": 5181,
              "end": 5209
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 5210,
                "end": 5215
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 5229,
                        "end": 5241
                      }
                    },
                    "loc": {
                      "start": 5229,
                      "end": 5241
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 5255,
                            "end": 5271
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5252,
                          "end": 5271
                        }
                      }
                    ],
                    "loc": {
                      "start": 5242,
                      "end": 5277
                    }
                  },
                  "loc": {
                    "start": 5222,
                    "end": 5277
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 5289,
                        "end": 5293
                      }
                    },
                    "loc": {
                      "start": 5289,
                      "end": 5293
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 5307,
                            "end": 5315
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5304,
                          "end": 5315
                        }
                      }
                    ],
                    "loc": {
                      "start": 5294,
                      "end": 5321
                    }
                  },
                  "loc": {
                    "start": 5282,
                    "end": 5321
                  }
                }
              ],
              "loc": {
                "start": 5216,
                "end": 5323
              }
            },
            "loc": {
              "start": 5210,
              "end": 5323
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 5324,
                "end": 5335
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5324,
              "end": 5335
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 5336,
                "end": 5350
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5336,
              "end": 5350
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 5351,
                "end": 5356
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5351,
              "end": 5356
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 5357,
                "end": 5366
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5357,
              "end": 5366
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 5367,
                "end": 5371
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 5381,
                      "end": 5389
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5378,
                    "end": 5389
                  }
                }
              ],
              "loc": {
                "start": 5372,
                "end": 5391
              }
            },
            "loc": {
              "start": 5367,
              "end": 5391
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 5392,
                "end": 5406
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5392,
              "end": 5406
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 5407,
                "end": 5412
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5407,
              "end": 5412
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5413,
                "end": 5416
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
                    "value": "canComment",
                    "loc": {
                      "start": 5423,
                      "end": 5433
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5423,
                    "end": 5433
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5438,
                      "end": 5447
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5438,
                    "end": 5447
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 5452,
                      "end": 5463
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5452,
                    "end": 5463
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5468,
                      "end": 5477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5468,
                    "end": 5477
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5482,
                      "end": 5489
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5482,
                    "end": 5489
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 5494,
                      "end": 5502
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5494,
                    "end": 5502
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 5507,
                      "end": 5519
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5507,
                    "end": 5519
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 5524,
                      "end": 5532
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5524,
                    "end": 5532
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 5537,
                      "end": 5545
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5537,
                    "end": 5545
                  }
                }
              ],
              "loc": {
                "start": 5417,
                "end": 5547
              }
            },
            "loc": {
              "start": 5413,
              "end": 5547
            }
          }
        ],
        "loc": {
          "start": 4603,
          "end": 5549
        }
      },
      "loc": {
        "start": 4570,
        "end": 5549
      }
    },
    "Routine_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_nav",
        "loc": {
          "start": 5559,
          "end": 5570
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 5574,
            "end": 5581
          }
        },
        "loc": {
          "start": 5574,
          "end": 5581
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
                "start": 5584,
                "end": 5586
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5584,
              "end": 5586
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5587,
                "end": 5597
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5587,
              "end": 5597
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5598,
                "end": 5607
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5598,
              "end": 5607
            }
          }
        ],
        "loc": {
          "start": 5582,
          "end": 5609
        }
      },
      "loc": {
        "start": 5550,
        "end": 5609
      }
    },
    "SmartContract_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "SmartContract_list",
        "loc": {
          "start": 5619,
          "end": 5637
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "SmartContract",
          "loc": {
            "start": 5641,
            "end": 5654
          }
        },
        "loc": {
          "start": 5641,
          "end": 5654
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
              "value": "versions",
              "loc": {
                "start": 5657,
                "end": 5665
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
                    "value": "translations",
                    "loc": {
                      "start": 5672,
                      "end": 5684
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
                            "start": 5695,
                            "end": 5697
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5695,
                          "end": 5697
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5706,
                            "end": 5714
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5706,
                          "end": 5714
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5723,
                            "end": 5734
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5723,
                          "end": 5734
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 5743,
                            "end": 5755
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5743,
                          "end": 5755
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5764,
                            "end": 5768
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5764,
                          "end": 5768
                        }
                      }
                    ],
                    "loc": {
                      "start": 5685,
                      "end": 5774
                    }
                  },
                  "loc": {
                    "start": 5672,
                    "end": 5774
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5779,
                      "end": 5781
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5779,
                    "end": 5781
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5786,
                      "end": 5796
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5786,
                    "end": 5796
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5801,
                      "end": 5811
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5801,
                    "end": 5811
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 5816,
                      "end": 5826
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5816,
                    "end": 5826
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 5831,
                      "end": 5840
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5831,
                    "end": 5840
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5845,
                      "end": 5853
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5845,
                    "end": 5853
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5858,
                      "end": 5867
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5858,
                    "end": 5867
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 5872,
                      "end": 5879
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5872,
                    "end": 5879
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contractType",
                    "loc": {
                      "start": 5884,
                      "end": 5896
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5884,
                    "end": 5896
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "content",
                    "loc": {
                      "start": 5901,
                      "end": 5908
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5901,
                    "end": 5908
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5913,
                      "end": 5925
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5913,
                    "end": 5925
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5930,
                      "end": 5942
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5930,
                    "end": 5942
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 5947,
                      "end": 5960
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5947,
                    "end": 5960
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 5965,
                      "end": 5987
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5965,
                    "end": 5987
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 5992,
                      "end": 6002
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5992,
                    "end": 6002
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 6007,
                      "end": 6019
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6007,
                    "end": 6019
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6024,
                      "end": 6027
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
                          "value": "canComment",
                          "loc": {
                            "start": 6038,
                            "end": 6048
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6038,
                          "end": 6048
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 6057,
                            "end": 6064
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6057,
                          "end": 6064
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 6073,
                            "end": 6082
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6073,
                          "end": 6082
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 6091,
                            "end": 6100
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6091,
                          "end": 6100
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6109,
                            "end": 6118
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6109,
                          "end": 6118
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 6127,
                            "end": 6133
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6127,
                          "end": 6133
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6142,
                            "end": 6149
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6142,
                          "end": 6149
                        }
                      }
                    ],
                    "loc": {
                      "start": 6028,
                      "end": 6155
                    }
                  },
                  "loc": {
                    "start": 6024,
                    "end": 6155
                  }
                }
              ],
              "loc": {
                "start": 5666,
                "end": 6157
              }
            },
            "loc": {
              "start": 5657,
              "end": 6157
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6158,
                "end": 6160
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6158,
              "end": 6160
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6161,
                "end": 6171
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6161,
              "end": 6171
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6172,
                "end": 6182
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6172,
              "end": 6182
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6183,
                "end": 6192
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6183,
              "end": 6192
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 6193,
                "end": 6204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6193,
              "end": 6204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 6205,
                "end": 6211
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
                    "value": "Label_list",
                    "loc": {
                      "start": 6221,
                      "end": 6231
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6218,
                    "end": 6231
                  }
                }
              ],
              "loc": {
                "start": 6212,
                "end": 6233
              }
            },
            "loc": {
              "start": 6205,
              "end": 6233
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 6234,
                "end": 6239
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 6253,
                        "end": 6265
                      }
                    },
                    "loc": {
                      "start": 6253,
                      "end": 6265
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 6279,
                            "end": 6295
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6276,
                          "end": 6295
                        }
                      }
                    ],
                    "loc": {
                      "start": 6266,
                      "end": 6301
                    }
                  },
                  "loc": {
                    "start": 6246,
                    "end": 6301
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 6313,
                        "end": 6317
                      }
                    },
                    "loc": {
                      "start": 6313,
                      "end": 6317
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 6331,
                            "end": 6339
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6328,
                          "end": 6339
                        }
                      }
                    ],
                    "loc": {
                      "start": 6318,
                      "end": 6345
                    }
                  },
                  "loc": {
                    "start": 6306,
                    "end": 6345
                  }
                }
              ],
              "loc": {
                "start": 6240,
                "end": 6347
              }
            },
            "loc": {
              "start": 6234,
              "end": 6347
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 6348,
                "end": 6359
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6348,
              "end": 6359
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 6360,
                "end": 6374
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6360,
              "end": 6374
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 6375,
                "end": 6380
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6375,
              "end": 6380
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6381,
                "end": 6390
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6381,
              "end": 6390
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 6391,
                "end": 6395
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 6405,
                      "end": 6413
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6402,
                    "end": 6413
                  }
                }
              ],
              "loc": {
                "start": 6396,
                "end": 6415
              }
            },
            "loc": {
              "start": 6391,
              "end": 6415
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 6416,
                "end": 6430
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6416,
              "end": 6430
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 6431,
                "end": 6436
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6431,
              "end": 6436
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6437,
                "end": 6440
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
                    "value": "canDelete",
                    "loc": {
                      "start": 6447,
                      "end": 6456
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6447,
                    "end": 6456
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 6461,
                      "end": 6472
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6461,
                    "end": 6472
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 6477,
                      "end": 6488
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6477,
                    "end": 6488
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6493,
                      "end": 6502
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6493,
                    "end": 6502
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6507,
                      "end": 6514
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6507,
                    "end": 6514
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 6519,
                      "end": 6527
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6519,
                    "end": 6527
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6532,
                      "end": 6544
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6532,
                    "end": 6544
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 6549,
                      "end": 6557
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6549,
                    "end": 6557
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 6562,
                      "end": 6570
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6562,
                    "end": 6570
                  }
                }
              ],
              "loc": {
                "start": 6441,
                "end": 6572
              }
            },
            "loc": {
              "start": 6437,
              "end": 6572
            }
          }
        ],
        "loc": {
          "start": 5655,
          "end": 6574
        }
      },
      "loc": {
        "start": 5610,
        "end": 6574
      }
    },
    "SmartContract_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "SmartContract_nav",
        "loc": {
          "start": 6584,
          "end": 6601
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "SmartContract",
          "loc": {
            "start": 6605,
            "end": 6618
          }
        },
        "loc": {
          "start": 6605,
          "end": 6618
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
                "start": 6621,
                "end": 6623
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6621,
              "end": 6623
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6624,
                "end": 6633
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6624,
              "end": 6633
            }
          }
        ],
        "loc": {
          "start": 6619,
          "end": 6635
        }
      },
      "loc": {
        "start": 6575,
        "end": 6635
      }
    },
    "Standard_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_list",
        "loc": {
          "start": 6645,
          "end": 6658
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 6662,
            "end": 6670
          }
        },
        "loc": {
          "start": 6662,
          "end": 6670
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
              "value": "versions",
              "loc": {
                "start": 6673,
                "end": 6681
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
                    "value": "translations",
                    "loc": {
                      "start": 6688,
                      "end": 6700
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
                            "start": 6711,
                            "end": 6713
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6711,
                          "end": 6713
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6722,
                            "end": 6730
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6722,
                          "end": 6730
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6739,
                            "end": 6750
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6739,
                          "end": 6750
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 6759,
                            "end": 6771
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6759,
                          "end": 6771
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6780,
                            "end": 6784
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6780,
                          "end": 6784
                        }
                      }
                    ],
                    "loc": {
                      "start": 6701,
                      "end": 6790
                    }
                  },
                  "loc": {
                    "start": 6688,
                    "end": 6790
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6795,
                      "end": 6797
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6795,
                    "end": 6797
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 6802,
                      "end": 6812
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6802,
                    "end": 6812
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 6817,
                      "end": 6827
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6817,
                    "end": 6827
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 6832,
                      "end": 6842
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6832,
                    "end": 6842
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isFile",
                    "loc": {
                      "start": 6847,
                      "end": 6853
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6847,
                    "end": 6853
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6858,
                      "end": 6866
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6858,
                    "end": 6866
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6871,
                      "end": 6880
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6871,
                    "end": 6880
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 6885,
                      "end": 6892
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6885,
                    "end": 6892
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardType",
                    "loc": {
                      "start": 6897,
                      "end": 6909
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6897,
                    "end": 6909
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "props",
                    "loc": {
                      "start": 6914,
                      "end": 6919
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6914,
                    "end": 6919
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yup",
                    "loc": {
                      "start": 6924,
                      "end": 6927
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6924,
                    "end": 6927
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6932,
                      "end": 6944
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6932,
                    "end": 6944
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6949,
                      "end": 6961
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6949,
                    "end": 6961
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 6966,
                      "end": 6979
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6966,
                    "end": 6979
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 6984,
                      "end": 7006
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6984,
                    "end": 7006
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 7011,
                      "end": 7021
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7011,
                    "end": 7021
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 7026,
                      "end": 7038
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7026,
                    "end": 7038
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 7043,
                      "end": 7046
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
                          "value": "canComment",
                          "loc": {
                            "start": 7057,
                            "end": 7067
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7057,
                          "end": 7067
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 7076,
                            "end": 7083
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7076,
                          "end": 7083
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 7092,
                            "end": 7101
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7092,
                          "end": 7101
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 7110,
                            "end": 7119
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7110,
                          "end": 7119
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 7128,
                            "end": 7137
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7128,
                          "end": 7137
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 7146,
                            "end": 7152
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7146,
                          "end": 7152
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 7161,
                            "end": 7168
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7161,
                          "end": 7168
                        }
                      }
                    ],
                    "loc": {
                      "start": 7047,
                      "end": 7174
                    }
                  },
                  "loc": {
                    "start": 7043,
                    "end": 7174
                  }
                }
              ],
              "loc": {
                "start": 6682,
                "end": 7176
              }
            },
            "loc": {
              "start": 6673,
              "end": 7176
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7177,
                "end": 7179
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7177,
              "end": 7179
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7180,
                "end": 7190
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7180,
              "end": 7190
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 7191,
                "end": 7201
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7191,
              "end": 7201
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7202,
                "end": 7211
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7202,
              "end": 7211
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 7212,
                "end": 7223
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7212,
              "end": 7223
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 7224,
                "end": 7230
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
                    "value": "Label_list",
                    "loc": {
                      "start": 7240,
                      "end": 7250
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7237,
                    "end": 7250
                  }
                }
              ],
              "loc": {
                "start": 7231,
                "end": 7252
              }
            },
            "loc": {
              "start": 7224,
              "end": 7252
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 7253,
                "end": 7258
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "Organization",
                      "loc": {
                        "start": 7272,
                        "end": 7284
                      }
                    },
                    "loc": {
                      "start": 7272,
                      "end": 7284
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Organization_nav",
                          "loc": {
                            "start": 7298,
                            "end": 7314
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7295,
                          "end": 7314
                        }
                      }
                    ],
                    "loc": {
                      "start": 7285,
                      "end": 7320
                    }
                  },
                  "loc": {
                    "start": 7265,
                    "end": 7320
                  }
                },
                {
                  "kind": "InlineFragment",
                  "typeCondition": {
                    "kind": "NamedType",
                    "name": {
                      "kind": "Name",
                      "value": "User",
                      "loc": {
                        "start": 7332,
                        "end": 7336
                      }
                    },
                    "loc": {
                      "start": 7332,
                      "end": 7336
                    }
                  },
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "User_nav",
                          "loc": {
                            "start": 7350,
                            "end": 7358
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7347,
                          "end": 7358
                        }
                      }
                    ],
                    "loc": {
                      "start": 7337,
                      "end": 7364
                    }
                  },
                  "loc": {
                    "start": 7325,
                    "end": 7364
                  }
                }
              ],
              "loc": {
                "start": 7259,
                "end": 7366
              }
            },
            "loc": {
              "start": 7253,
              "end": 7366
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 7367,
                "end": 7378
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7367,
              "end": 7378
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 7379,
                "end": 7393
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7379,
              "end": 7393
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 7394,
                "end": 7399
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7394,
              "end": 7399
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7400,
                "end": 7409
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7400,
              "end": 7409
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 7410,
                "end": 7414
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
                    "value": "Tag_list",
                    "loc": {
                      "start": 7424,
                      "end": 7432
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7421,
                    "end": 7432
                  }
                }
              ],
              "loc": {
                "start": 7415,
                "end": 7434
              }
            },
            "loc": {
              "start": 7410,
              "end": 7434
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 7435,
                "end": 7449
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7435,
              "end": 7449
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 7450,
                "end": 7455
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7450,
              "end": 7455
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7456,
                "end": 7459
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
                    "value": "canDelete",
                    "loc": {
                      "start": 7466,
                      "end": 7475
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7466,
                    "end": 7475
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 7480,
                      "end": 7491
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7480,
                    "end": 7491
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 7496,
                      "end": 7507
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7496,
                    "end": 7507
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7512,
                      "end": 7521
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7512,
                    "end": 7521
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7526,
                      "end": 7533
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7526,
                    "end": 7533
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 7538,
                      "end": 7546
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7538,
                    "end": 7546
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7551,
                      "end": 7563
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7551,
                    "end": 7563
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7568,
                      "end": 7576
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7568,
                    "end": 7576
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 7581,
                      "end": 7589
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7581,
                    "end": 7589
                  }
                }
              ],
              "loc": {
                "start": 7460,
                "end": 7591
              }
            },
            "loc": {
              "start": 7456,
              "end": 7591
            }
          }
        ],
        "loc": {
          "start": 6671,
          "end": 7593
        }
      },
      "loc": {
        "start": 6636,
        "end": 7593
      }
    },
    "Standard_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_nav",
        "loc": {
          "start": 7603,
          "end": 7615
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 7619,
            "end": 7627
          }
        },
        "loc": {
          "start": 7619,
          "end": 7627
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
                "start": 7630,
                "end": 7632
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7630,
              "end": 7632
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7633,
                "end": 7642
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7633,
              "end": 7642
            }
          }
        ],
        "loc": {
          "start": 7628,
          "end": 7644
        }
      },
      "loc": {
        "start": 7594,
        "end": 7644
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 7654,
          "end": 7662
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 7666,
            "end": 7669
          }
        },
        "loc": {
          "start": 7666,
          "end": 7669
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
                "start": 7672,
                "end": 7674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7672,
              "end": 7674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7675,
                "end": 7685
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7675,
              "end": 7685
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 7686,
                "end": 7689
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7686,
              "end": 7689
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7690,
                "end": 7699
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7690,
              "end": 7699
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 7700,
                "end": 7712
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
                      "start": 7719,
                      "end": 7721
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7719,
                    "end": 7721
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7726,
                      "end": 7734
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7726,
                    "end": 7734
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 7739,
                      "end": 7750
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7739,
                    "end": 7750
                  }
                }
              ],
              "loc": {
                "start": 7713,
                "end": 7752
              }
            },
            "loc": {
              "start": 7700,
              "end": 7752
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7753,
                "end": 7756
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
                    "value": "isOwn",
                    "loc": {
                      "start": 7763,
                      "end": 7768
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7763,
                    "end": 7768
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7773,
                      "end": 7785
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7773,
                    "end": 7785
                  }
                }
              ],
              "loc": {
                "start": 7757,
                "end": 7787
              }
            },
            "loc": {
              "start": 7753,
              "end": 7787
            }
          }
        ],
        "loc": {
          "start": 7670,
          "end": 7789
        }
      },
      "loc": {
        "start": 7645,
        "end": 7789
      }
    },
    "User_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_list",
        "loc": {
          "start": 7799,
          "end": 7808
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7812,
            "end": 7816
          }
        },
        "loc": {
          "start": 7812,
          "end": 7816
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
              "value": "translations",
              "loc": {
                "start": 7819,
                "end": 7831
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
                      "start": 7838,
                      "end": 7840
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7838,
                    "end": 7840
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7845,
                      "end": 7853
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7845,
                    "end": 7853
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 7858,
                      "end": 7861
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7858,
                    "end": 7861
                  }
                }
              ],
              "loc": {
                "start": 7832,
                "end": 7863
              }
            },
            "loc": {
              "start": 7819,
              "end": 7863
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7864,
                "end": 7866
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7864,
              "end": 7866
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7867,
                "end": 7877
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7867,
              "end": 7877
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7878,
                "end": 7884
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7878,
              "end": 7884
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7885,
                "end": 7890
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7885,
              "end": 7890
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7891,
                "end": 7895
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7891,
              "end": 7895
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7896,
                "end": 7905
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7896,
              "end": 7905
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsReceivedCount",
              "loc": {
                "start": 7906,
                "end": 7926
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7906,
              "end": 7926
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7927,
                "end": 7930
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
                    "value": "canDelete",
                    "loc": {
                      "start": 7937,
                      "end": 7946
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7937,
                    "end": 7946
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7951,
                      "end": 7960
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7951,
                    "end": 7960
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7965,
                      "end": 7974
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7965,
                    "end": 7974
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7979,
                      "end": 7991
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7979,
                    "end": 7991
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7996,
                      "end": 8004
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7996,
                    "end": 8004
                  }
                }
              ],
              "loc": {
                "start": 7931,
                "end": 8006
              }
            },
            "loc": {
              "start": 7927,
              "end": 8006
            }
          }
        ],
        "loc": {
          "start": 7817,
          "end": 8008
        }
      },
      "loc": {
        "start": 7790,
        "end": 8008
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 8018,
          "end": 8026
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 8030,
            "end": 8034
          }
        },
        "loc": {
          "start": 8030,
          "end": 8034
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
                "start": 8037,
                "end": 8039
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8037,
              "end": 8039
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 8040,
                "end": 8045
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8040,
              "end": 8045
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 8046,
                "end": 8050
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8046,
              "end": 8050
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 8051,
                "end": 8057
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8051,
              "end": 8057
            }
          }
        ],
        "loc": {
          "start": 8035,
          "end": 8059
        }
      },
      "loc": {
        "start": 8009,
        "end": 8059
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "popular",
      "loc": {
        "start": 8067,
        "end": 8074
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
              "start": 8076,
              "end": 8081
            }
          },
          "loc": {
            "start": 8075,
            "end": 8081
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "PopularInput",
              "loc": {
                "start": 8083,
                "end": 8095
              }
            },
            "loc": {
              "start": 8083,
              "end": 8095
            }
          },
          "loc": {
            "start": 8083,
            "end": 8096
          }
        },
        "directives": [],
        "loc": {
          "start": 8075,
          "end": 8096
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
            "value": "popular",
            "loc": {
              "start": 8102,
              "end": 8109
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 8110,
                  "end": 8115
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 8118,
                    "end": 8123
                  }
                },
                "loc": {
                  "start": 8117,
                  "end": 8123
                }
              },
              "loc": {
                "start": 8110,
                "end": 8123
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
                  "value": "apis",
                  "loc": {
                    "start": 8131,
                    "end": 8135
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
                        "value": "Api_list",
                        "loc": {
                          "start": 8149,
                          "end": 8157
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8146,
                        "end": 8157
                      }
                    }
                  ],
                  "loc": {
                    "start": 8136,
                    "end": 8163
                  }
                },
                "loc": {
                  "start": 8131,
                  "end": 8163
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "notes",
                  "loc": {
                    "start": 8168,
                    "end": 8173
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
                        "value": "Note_list",
                        "loc": {
                          "start": 8187,
                          "end": 8196
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8184,
                        "end": 8196
                      }
                    }
                  ],
                  "loc": {
                    "start": 8174,
                    "end": 8202
                  }
                },
                "loc": {
                  "start": 8168,
                  "end": 8202
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "organizations",
                  "loc": {
                    "start": 8207,
                    "end": 8220
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
                        "value": "Organization_list",
                        "loc": {
                          "start": 8234,
                          "end": 8251
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8231,
                        "end": 8251
                      }
                    }
                  ],
                  "loc": {
                    "start": 8221,
                    "end": 8257
                  }
                },
                "loc": {
                  "start": 8207,
                  "end": 8257
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "projects",
                  "loc": {
                    "start": 8262,
                    "end": 8270
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
                        "value": "Project_list",
                        "loc": {
                          "start": 8284,
                          "end": 8296
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8281,
                        "end": 8296
                      }
                    }
                  ],
                  "loc": {
                    "start": 8271,
                    "end": 8302
                  }
                },
                "loc": {
                  "start": 8262,
                  "end": 8302
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "questions",
                  "loc": {
                    "start": 8307,
                    "end": 8316
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
                        "value": "Question_list",
                        "loc": {
                          "start": 8330,
                          "end": 8343
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8327,
                        "end": 8343
                      }
                    }
                  ],
                  "loc": {
                    "start": 8317,
                    "end": 8349
                  }
                },
                "loc": {
                  "start": 8307,
                  "end": 8349
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "routines",
                  "loc": {
                    "start": 8354,
                    "end": 8362
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
                        "value": "Routine_list",
                        "loc": {
                          "start": 8376,
                          "end": 8388
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8373,
                        "end": 8388
                      }
                    }
                  ],
                  "loc": {
                    "start": 8363,
                    "end": 8394
                  }
                },
                "loc": {
                  "start": 8354,
                  "end": 8394
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "smartContracts",
                  "loc": {
                    "start": 8399,
                    "end": 8413
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
                        "value": "SmartContract_list",
                        "loc": {
                          "start": 8427,
                          "end": 8445
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8424,
                        "end": 8445
                      }
                    }
                  ],
                  "loc": {
                    "start": 8414,
                    "end": 8451
                  }
                },
                "loc": {
                  "start": 8399,
                  "end": 8451
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "standards",
                  "loc": {
                    "start": 8456,
                    "end": 8465
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
                        "value": "Standard_list",
                        "loc": {
                          "start": 8479,
                          "end": 8492
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8476,
                        "end": 8492
                      }
                    }
                  ],
                  "loc": {
                    "start": 8466,
                    "end": 8498
                  }
                },
                "loc": {
                  "start": 8456,
                  "end": 8498
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "users",
                  "loc": {
                    "start": 8503,
                    "end": 8508
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
                        "value": "User_list",
                        "loc": {
                          "start": 8522,
                          "end": 8531
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8519,
                        "end": 8531
                      }
                    }
                  ],
                  "loc": {
                    "start": 8509,
                    "end": 8537
                  }
                },
                "loc": {
                  "start": 8503,
                  "end": 8537
                }
              }
            ],
            "loc": {
              "start": 8125,
              "end": 8541
            }
          },
          "loc": {
            "start": 8102,
            "end": 8541
          }
        }
      ],
      "loc": {
        "start": 8098,
        "end": 8543
      }
    },
    "loc": {
      "start": 8061,
      "end": 8543
    }
  },
  "variableValues": {},
  "path": {
    "key": "feed_popular"
  }
} as const;
