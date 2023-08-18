export const popular_findMany = {
  "fieldName": "populars",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "populars",
        "loc": {
          "start": 7882,
          "end": 7890
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 7891,
              "end": 7896
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 7899,
                "end": 7904
              }
            },
            "loc": {
              "start": 7898,
              "end": 7904
            }
          },
          "loc": {
            "start": 7891,
            "end": 7904
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
                "start": 7912,
                "end": 7917
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
                      "start": 7928,
                      "end": 7934
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7928,
                    "end": 7934
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "node",
                    "loc": {
                      "start": 7943,
                      "end": 7947
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
                              "start": 7969,
                              "end": 7972
                            }
                          },
                          "loc": {
                            "start": 7969,
                            "end": 7972
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
                                "value": "Api_list",
                                "loc": {
                                  "start": 7994,
                                  "end": 8002
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 7991,
                                "end": 8002
                              }
                            }
                          ],
                          "loc": {
                            "start": 7973,
                            "end": 8016
                          }
                        },
                        "loc": {
                          "start": 7962,
                          "end": 8016
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
                              "start": 8036,
                              "end": 8040
                            }
                          },
                          "loc": {
                            "start": 8036,
                            "end": 8040
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
                                "value": "Note_list",
                                "loc": {
                                  "start": 8062,
                                  "end": 8071
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8059,
                                "end": 8071
                              }
                            }
                          ],
                          "loc": {
                            "start": 8041,
                            "end": 8085
                          }
                        },
                        "loc": {
                          "start": 8029,
                          "end": 8085
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
                              "start": 8105,
                              "end": 8117
                            }
                          },
                          "loc": {
                            "start": 8105,
                            "end": 8117
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
                                "value": "Organization_list",
                                "loc": {
                                  "start": 8139,
                                  "end": 8156
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8136,
                                "end": 8156
                              }
                            }
                          ],
                          "loc": {
                            "start": 8118,
                            "end": 8170
                          }
                        },
                        "loc": {
                          "start": 8098,
                          "end": 8170
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
                              "start": 8190,
                              "end": 8197
                            }
                          },
                          "loc": {
                            "start": 8190,
                            "end": 8197
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
                                "value": "Project_list",
                                "loc": {
                                  "start": 8219,
                                  "end": 8231
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8216,
                                "end": 8231
                              }
                            }
                          ],
                          "loc": {
                            "start": 8198,
                            "end": 8245
                          }
                        },
                        "loc": {
                          "start": 8183,
                          "end": 8245
                        }
                      },
                      {
                        "kind": "InlineFragment",
                        "typeCondition": {
                          "kind": "NamedType",
                          "name": {
                            "kind": "Name",
                            "value": "Question",
                            "loc": {
                              "start": 8265,
                              "end": 8273
                            }
                          },
                          "loc": {
                            "start": 8265,
                            "end": 8273
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
                                "value": "Question_list",
                                "loc": {
                                  "start": 8295,
                                  "end": 8308
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8292,
                                "end": 8308
                              }
                            }
                          ],
                          "loc": {
                            "start": 8274,
                            "end": 8322
                          }
                        },
                        "loc": {
                          "start": 8258,
                          "end": 8322
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
                              "start": 8342,
                              "end": 8349
                            }
                          },
                          "loc": {
                            "start": 8342,
                            "end": 8349
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
                                "value": "Routine_list",
                                "loc": {
                                  "start": 8371,
                                  "end": 8383
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8368,
                                "end": 8383
                              }
                            }
                          ],
                          "loc": {
                            "start": 8350,
                            "end": 8397
                          }
                        },
                        "loc": {
                          "start": 8335,
                          "end": 8397
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
                              "start": 8417,
                              "end": 8430
                            }
                          },
                          "loc": {
                            "start": 8417,
                            "end": 8430
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
                                "value": "SmartContract_list",
                                "loc": {
                                  "start": 8452,
                                  "end": 8470
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8449,
                                "end": 8470
                              }
                            }
                          ],
                          "loc": {
                            "start": 8431,
                            "end": 8484
                          }
                        },
                        "loc": {
                          "start": 8410,
                          "end": 8484
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
                              "start": 8504,
                              "end": 8512
                            }
                          },
                          "loc": {
                            "start": 8504,
                            "end": 8512
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
                                "value": "Standard_list",
                                "loc": {
                                  "start": 8534,
                                  "end": 8547
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8531,
                                "end": 8547
                              }
                            }
                          ],
                          "loc": {
                            "start": 8513,
                            "end": 8561
                          }
                        },
                        "loc": {
                          "start": 8497,
                          "end": 8561
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
                              "start": 8581,
                              "end": 8585
                            }
                          },
                          "loc": {
                            "start": 8581,
                            "end": 8585
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
                                "value": "User_list",
                                "loc": {
                                  "start": 8607,
                                  "end": 8616
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8604,
                                "end": 8616
                              }
                            }
                          ],
                          "loc": {
                            "start": 8586,
                            "end": 8630
                          }
                        },
                        "loc": {
                          "start": 8574,
                          "end": 8630
                        }
                      }
                    ],
                    "loc": {
                      "start": 7948,
                      "end": 8640
                    }
                  },
                  "loc": {
                    "start": 7943,
                    "end": 8640
                  }
                }
              ],
              "loc": {
                "start": 7918,
                "end": 8646
              }
            },
            "loc": {
              "start": 7912,
              "end": 8646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "pageInfo",
              "loc": {
                "start": 8651,
                "end": 8659
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
                    "value": "hasNextPage",
                    "loc": {
                      "start": 8670,
                      "end": 8681
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8670,
                    "end": 8681
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorApi",
                    "loc": {
                      "start": 8690,
                      "end": 8702
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8690,
                    "end": 8702
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorNote",
                    "loc": {
                      "start": 8711,
                      "end": 8724
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8711,
                    "end": 8724
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorOrganization",
                    "loc": {
                      "start": 8733,
                      "end": 8754
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8733,
                    "end": 8754
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorProject",
                    "loc": {
                      "start": 8763,
                      "end": 8779
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8763,
                    "end": 8779
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorQuestion",
                    "loc": {
                      "start": 8788,
                      "end": 8805
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8788,
                    "end": 8805
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorRoutine",
                    "loc": {
                      "start": 8814,
                      "end": 8830
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8814,
                    "end": 8830
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorSmartContract",
                    "loc": {
                      "start": 8839,
                      "end": 8861
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8839,
                    "end": 8861
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorStandard",
                    "loc": {
                      "start": 8870,
                      "end": 8887
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8870,
                    "end": 8887
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endCursorUser",
                    "loc": {
                      "start": 8896,
                      "end": 8909
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8896,
                    "end": 8909
                  }
                }
              ],
              "loc": {
                "start": 8660,
                "end": 8915
              }
            },
            "loc": {
              "start": 8651,
              "end": 8915
            }
          }
        ],
        "loc": {
          "start": 7906,
          "end": 8919
        }
      },
      "loc": {
        "start": 7882,
        "end": 8919
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 28,
          "end": 36
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
                "start": 43,
                "end": 55
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
                      "start": 66,
                      "end": 68
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 66,
                    "end": 68
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 77,
                      "end": 85
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 77,
                    "end": 85
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "details",
                    "loc": {
                      "start": 94,
                      "end": 101
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 94,
                    "end": 101
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 110,
                      "end": 114
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 110,
                    "end": 114
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "summary",
                    "loc": {
                      "start": 123,
                      "end": 130
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 123,
                    "end": 130
                  }
                }
              ],
              "loc": {
                "start": 56,
                "end": 136
              }
            },
            "loc": {
              "start": 43,
              "end": 136
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 141,
                "end": 143
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 141,
              "end": 143
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 148,
                "end": 158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 148,
              "end": 158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 163,
                "end": 173
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 163,
              "end": 173
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "callLink",
              "loc": {
                "start": 178,
                "end": 186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 178,
              "end": 186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 191,
                "end": 204
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 191,
              "end": 204
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "documentationLink",
              "loc": {
                "start": 209,
                "end": 226
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 209,
              "end": 226
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 231,
                "end": 241
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 231,
              "end": 241
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 246,
                "end": 254
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 246,
              "end": 254
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
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
              "value": "reportsCount",
              "loc": {
                "start": 273,
                "end": 285
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 273,
              "end": 285
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 290,
                "end": 302
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 290,
              "end": 302
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 307,
                "end": 319
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 307,
              "end": 319
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 324,
                "end": 327
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
                      "start": 338,
                      "end": 348
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 338,
                    "end": 348
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 357,
                      "end": 364
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 357,
                    "end": 364
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 373,
                      "end": 382
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 373,
                    "end": 382
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 391,
                      "end": 400
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 391,
                    "end": 400
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 409,
                      "end": 418
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 409,
                    "end": 418
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 427,
                      "end": 433
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 427,
                    "end": 433
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 442,
                      "end": 449
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 442,
                    "end": 449
                  }
                }
              ],
              "loc": {
                "start": 328,
                "end": 455
              }
            },
            "loc": {
              "start": 324,
              "end": 455
            }
          }
        ],
        "loc": {
          "start": 37,
          "end": 457
        }
      },
      "loc": {
        "start": 28,
        "end": 457
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 458,
          "end": 460
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 458,
        "end": 460
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 461,
          "end": 471
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 461,
        "end": 471
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 472,
          "end": 482
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 472,
        "end": 482
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 483,
          "end": 492
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 483,
        "end": 492
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 493,
          "end": 504
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 493,
        "end": 504
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 505,
          "end": 511
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
                "start": 521,
                "end": 531
              }
            },
            "directives": [],
            "loc": {
              "start": 518,
              "end": 531
            }
          }
        ],
        "loc": {
          "start": 512,
          "end": 533
        }
      },
      "loc": {
        "start": 505,
        "end": 533
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 534,
          "end": 539
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
                  "start": 553,
                  "end": 565
                }
              },
              "loc": {
                "start": 553,
                "end": 565
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
                      "start": 579,
                      "end": 595
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 576,
                    "end": 595
                  }
                }
              ],
              "loc": {
                "start": 566,
                "end": 601
              }
            },
            "loc": {
              "start": 546,
              "end": 601
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
                  "start": 613,
                  "end": 617
                }
              },
              "loc": {
                "start": 613,
                "end": 617
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
                      "start": 631,
                      "end": 639
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 628,
                    "end": 639
                  }
                }
              ],
              "loc": {
                "start": 618,
                "end": 645
              }
            },
            "loc": {
              "start": 606,
              "end": 645
            }
          }
        ],
        "loc": {
          "start": 540,
          "end": 647
        }
      },
      "loc": {
        "start": 534,
        "end": 647
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 648,
          "end": 659
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 648,
        "end": 659
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 660,
          "end": 674
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 660,
        "end": 674
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 675,
          "end": 680
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 675,
        "end": 680
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 681,
          "end": 690
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 681,
        "end": 690
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 691,
          "end": 695
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
                "start": 705,
                "end": 713
              }
            },
            "directives": [],
            "loc": {
              "start": 702,
              "end": 713
            }
          }
        ],
        "loc": {
          "start": 696,
          "end": 715
        }
      },
      "loc": {
        "start": 691,
        "end": 715
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 716,
          "end": 730
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 716,
        "end": 730
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 731,
          "end": 736
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 731,
        "end": 736
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 737,
          "end": 740
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
                "start": 747,
                "end": 756
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 747,
              "end": 756
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 761,
                "end": 772
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 761,
              "end": 772
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 777,
                "end": 788
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 777,
              "end": 788
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 793,
                "end": 802
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 793,
              "end": 802
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 807,
                "end": 814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 807,
              "end": 814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 819,
                "end": 827
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 819,
              "end": 827
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 832,
                "end": 844
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 832,
              "end": 844
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 849,
                "end": 857
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 849,
              "end": 857
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 862,
                "end": 870
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 862,
              "end": 870
            }
          }
        ],
        "loc": {
          "start": 741,
          "end": 872
        }
      },
      "loc": {
        "start": 737,
        "end": 872
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 901,
          "end": 903
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 901,
        "end": 903
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 904,
          "end": 913
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 904,
        "end": 913
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 947,
          "end": 949
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 947,
        "end": 949
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 950,
          "end": 960
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 950,
        "end": 960
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 961,
          "end": 971
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 961,
        "end": 971
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "color",
        "loc": {
          "start": 972,
          "end": 977
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 972,
        "end": 977
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "label",
        "loc": {
          "start": 978,
          "end": 983
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 978,
        "end": 983
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 984,
          "end": 989
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
                  "start": 1003,
                  "end": 1015
                }
              },
              "loc": {
                "start": 1003,
                "end": 1015
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
                      "start": 1029,
                      "end": 1045
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1026,
                    "end": 1045
                  }
                }
              ],
              "loc": {
                "start": 1016,
                "end": 1051
              }
            },
            "loc": {
              "start": 996,
              "end": 1051
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
                  "start": 1063,
                  "end": 1067
                }
              },
              "loc": {
                "start": 1063,
                "end": 1067
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
                      "start": 1081,
                      "end": 1089
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1078,
                    "end": 1089
                  }
                }
              ],
              "loc": {
                "start": 1068,
                "end": 1095
              }
            },
            "loc": {
              "start": 1056,
              "end": 1095
            }
          }
        ],
        "loc": {
          "start": 990,
          "end": 1097
        }
      },
      "loc": {
        "start": 984,
        "end": 1097
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1098,
          "end": 1101
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
                "start": 1108,
                "end": 1117
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1108,
              "end": 1117
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1122,
                "end": 1131
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1122,
              "end": 1131
            }
          }
        ],
        "loc": {
          "start": 1102,
          "end": 1133
        }
      },
      "loc": {
        "start": 1098,
        "end": 1133
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 1165,
          "end": 1173
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
                "start": 1180,
                "end": 1192
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
                      "start": 1203,
                      "end": 1205
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1203,
                    "end": 1205
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 1214,
                      "end": 1222
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1214,
                    "end": 1222
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 1231,
                      "end": 1242
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1231,
                    "end": 1242
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 1251,
                      "end": 1255
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1251,
                    "end": 1255
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "text",
                    "loc": {
                      "start": 1264,
                      "end": 1268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1264,
                    "end": 1268
                  }
                }
              ],
              "loc": {
                "start": 1193,
                "end": 1274
              }
            },
            "loc": {
              "start": 1180,
              "end": 1274
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1279,
                "end": 1281
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1279,
              "end": 1281
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1286,
                "end": 1296
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1286,
              "end": 1296
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1301,
                "end": 1311
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1301,
              "end": 1311
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 1316,
                "end": 1324
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1316,
              "end": 1324
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1329,
                "end": 1338
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1329,
              "end": 1338
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 1343,
                "end": 1355
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1343,
              "end": 1355
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 1360,
                "end": 1372
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1360,
              "end": 1372
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 1377,
                "end": 1389
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1377,
              "end": 1389
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1394,
                "end": 1397
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
                      "start": 1408,
                      "end": 1418
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1408,
                    "end": 1418
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 1427,
                      "end": 1434
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1427,
                    "end": 1434
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 1443,
                      "end": 1452
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1443,
                    "end": 1452
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 1461,
                      "end": 1470
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1461,
                    "end": 1470
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1479,
                      "end": 1488
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1479,
                    "end": 1488
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 1497,
                      "end": 1503
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1497,
                    "end": 1503
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1512,
                      "end": 1519
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1512,
                    "end": 1519
                  }
                }
              ],
              "loc": {
                "start": 1398,
                "end": 1525
              }
            },
            "loc": {
              "start": 1394,
              "end": 1525
            }
          }
        ],
        "loc": {
          "start": 1174,
          "end": 1527
        }
      },
      "loc": {
        "start": 1165,
        "end": 1527
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1528,
          "end": 1530
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1528,
        "end": 1530
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 1531,
          "end": 1541
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1531,
        "end": 1541
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 1542,
          "end": 1552
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1542,
        "end": 1552
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1553,
          "end": 1562
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1553,
        "end": 1562
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 1563,
          "end": 1574
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1563,
        "end": 1574
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 1575,
          "end": 1581
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
                "start": 1591,
                "end": 1601
              }
            },
            "directives": [],
            "loc": {
              "start": 1588,
              "end": 1601
            }
          }
        ],
        "loc": {
          "start": 1582,
          "end": 1603
        }
      },
      "loc": {
        "start": 1575,
        "end": 1603
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 1604,
          "end": 1609
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
                  "start": 1623,
                  "end": 1635
                }
              },
              "loc": {
                "start": 1623,
                "end": 1635
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
                      "start": 1649,
                      "end": 1665
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1646,
                    "end": 1665
                  }
                }
              ],
              "loc": {
                "start": 1636,
                "end": 1671
              }
            },
            "loc": {
              "start": 1616,
              "end": 1671
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
                  "start": 1683,
                  "end": 1687
                }
              },
              "loc": {
                "start": 1683,
                "end": 1687
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
                      "start": 1701,
                      "end": 1709
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1698,
                    "end": 1709
                  }
                }
              ],
              "loc": {
                "start": 1688,
                "end": 1715
              }
            },
            "loc": {
              "start": 1676,
              "end": 1715
            }
          }
        ],
        "loc": {
          "start": 1610,
          "end": 1717
        }
      },
      "loc": {
        "start": 1604,
        "end": 1717
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 1718,
          "end": 1729
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1718,
        "end": 1729
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 1730,
          "end": 1744
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1730,
        "end": 1744
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 1745,
          "end": 1750
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1745,
        "end": 1750
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 1751,
          "end": 1760
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1751,
        "end": 1760
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 1761,
          "end": 1765
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
                "start": 1775,
                "end": 1783
              }
            },
            "directives": [],
            "loc": {
              "start": 1772,
              "end": 1783
            }
          }
        ],
        "loc": {
          "start": 1766,
          "end": 1785
        }
      },
      "loc": {
        "start": 1761,
        "end": 1785
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 1786,
          "end": 1800
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1786,
        "end": 1800
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 1801,
          "end": 1806
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1801,
        "end": 1806
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 1807,
          "end": 1810
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
                "start": 1817,
                "end": 1826
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1817,
              "end": 1826
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 1831,
                "end": 1842
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1831,
              "end": 1842
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 1847,
                "end": 1858
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1847,
              "end": 1858
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 1863,
                "end": 1872
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1863,
              "end": 1872
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 1877,
                "end": 1884
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1877,
              "end": 1884
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 1889,
                "end": 1897
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1889,
              "end": 1897
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 1902,
                "end": 1914
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1902,
              "end": 1914
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 1919,
                "end": 1927
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1919,
              "end": 1927
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 1932,
                "end": 1940
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1932,
              "end": 1940
            }
          }
        ],
        "loc": {
          "start": 1811,
          "end": 1942
        }
      },
      "loc": {
        "start": 1807,
        "end": 1942
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 1973,
          "end": 1975
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1973,
        "end": 1975
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 1976,
          "end": 1985
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 1976,
        "end": 1985
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2033,
          "end": 2035
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2033,
        "end": 2035
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2036,
          "end": 2047
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2036,
        "end": 2047
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2048,
          "end": 2054
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2048,
        "end": 2054
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 2055,
          "end": 2065
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2055,
        "end": 2065
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 2066,
          "end": 2076
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2066,
        "end": 2076
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isOpenToNewMembers",
        "loc": {
          "start": 2077,
          "end": 2095
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2077,
        "end": 2095
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 2096,
          "end": 2105
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2096,
        "end": 2105
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 2106,
          "end": 2119
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2106,
        "end": 2119
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "membersCount",
        "loc": {
          "start": 2120,
          "end": 2132
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2120,
        "end": 2132
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2133,
          "end": 2145
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2133,
        "end": 2145
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 2146,
          "end": 2158
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2146,
        "end": 2158
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 2159,
          "end": 2168
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2159,
        "end": 2168
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 2169,
          "end": 2173
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
                "start": 2183,
                "end": 2191
              }
            },
            "directives": [],
            "loc": {
              "start": 2180,
              "end": 2191
            }
          }
        ],
        "loc": {
          "start": 2174,
          "end": 2193
        }
      },
      "loc": {
        "start": 2169,
        "end": 2193
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 2194,
          "end": 2206
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
                "start": 2213,
                "end": 2215
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2213,
              "end": 2215
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 2220,
                "end": 2228
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2220,
              "end": 2228
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bio",
              "loc": {
                "start": 2233,
                "end": 2236
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2233,
              "end": 2236
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 2241,
                "end": 2245
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2241,
              "end": 2245
            }
          }
        ],
        "loc": {
          "start": 2207,
          "end": 2247
        }
      },
      "loc": {
        "start": 2194,
        "end": 2247
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2248,
          "end": 2251
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
                "start": 2258,
                "end": 2271
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2258,
              "end": 2271
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2276,
                "end": 2285
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2276,
              "end": 2285
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2290,
                "end": 2301
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2290,
              "end": 2301
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 2306,
                "end": 2315
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2306,
              "end": 2315
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2320,
                "end": 2329
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2320,
              "end": 2329
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2334,
                "end": 2341
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2334,
              "end": 2341
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2346,
                "end": 2358
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2346,
              "end": 2358
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2363,
                "end": 2371
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2363,
              "end": 2371
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 2376,
                "end": 2390
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
                      "start": 2401,
                      "end": 2403
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2401,
                    "end": 2403
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2412,
                      "end": 2422
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2412,
                    "end": 2422
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2431,
                      "end": 2441
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2431,
                    "end": 2441
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 2450,
                      "end": 2457
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2450,
                    "end": 2457
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 2466,
                      "end": 2477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2466,
                    "end": 2477
                  }
                }
              ],
              "loc": {
                "start": 2391,
                "end": 2483
              }
            },
            "loc": {
              "start": 2376,
              "end": 2483
            }
          }
        ],
        "loc": {
          "start": 2252,
          "end": 2485
        }
      },
      "loc": {
        "start": 2248,
        "end": 2485
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 2532,
          "end": 2534
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2532,
        "end": 2534
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 2535,
          "end": 2546
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2535,
        "end": 2546
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 2547,
          "end": 2553
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2547,
        "end": 2553
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 2554,
          "end": 2566
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 2554,
        "end": 2566
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 2567,
          "end": 2570
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
                "start": 2577,
                "end": 2590
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2577,
              "end": 2590
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 2595,
                "end": 2604
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2595,
              "end": 2604
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 2609,
                "end": 2620
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2609,
              "end": 2620
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 2625,
                "end": 2634
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2625,
              "end": 2634
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 2639,
                "end": 2648
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2639,
              "end": 2648
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 2653,
                "end": 2660
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2653,
              "end": 2660
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 2665,
                "end": 2677
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2665,
              "end": 2677
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 2682,
                "end": 2690
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2682,
              "end": 2690
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yourMembership",
              "loc": {
                "start": 2695,
                "end": 2709
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
                      "start": 2720,
                      "end": 2722
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2720,
                    "end": 2722
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2731,
                      "end": 2741
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2731,
                    "end": 2741
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2750,
                      "end": 2760
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2750,
                    "end": 2760
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAdmin",
                    "loc": {
                      "start": 2769,
                      "end": 2776
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2769,
                    "end": 2776
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "permissions",
                    "loc": {
                      "start": 2785,
                      "end": 2796
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2785,
                    "end": 2796
                  }
                }
              ],
              "loc": {
                "start": 2710,
                "end": 2802
              }
            },
            "loc": {
              "start": 2695,
              "end": 2802
            }
          }
        ],
        "loc": {
          "start": 2571,
          "end": 2804
        }
      },
      "loc": {
        "start": 2567,
        "end": 2804
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 2842,
          "end": 2850
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
                "start": 2857,
                "end": 2869
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
                      "start": 2880,
                      "end": 2882
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2880,
                    "end": 2882
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2891,
                      "end": 2899
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2891,
                    "end": 2899
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 2908,
                      "end": 2919
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2908,
                    "end": 2919
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2928,
                      "end": 2932
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2928,
                    "end": 2932
                  }
                }
              ],
              "loc": {
                "start": 2870,
                "end": 2938
              }
            },
            "loc": {
              "start": 2857,
              "end": 2938
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 2943,
                "end": 2945
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2943,
              "end": 2945
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2950,
                "end": 2960
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2950,
              "end": 2960
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2965,
                "end": 2975
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2965,
              "end": 2975
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoriesCount",
              "loc": {
                "start": 2980,
                "end": 2996
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2980,
              "end": 2996
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 3001,
                "end": 3009
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3001,
              "end": 3009
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3014,
                "end": 3023
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3014,
              "end": 3023
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3028,
                "end": 3040
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3028,
              "end": 3040
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "runProjectsCount",
              "loc": {
                "start": 3045,
                "end": 3061
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3045,
              "end": 3061
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 3066,
                "end": 3076
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3066,
              "end": 3076
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 3081,
                "end": 3093
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3081,
              "end": 3093
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 3098,
                "end": 3110
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3098,
              "end": 3110
            }
          }
        ],
        "loc": {
          "start": 2851,
          "end": 3112
        }
      },
      "loc": {
        "start": 2842,
        "end": 3112
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3113,
          "end": 3115
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3113,
        "end": 3115
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3116,
          "end": 3126
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3116,
        "end": 3126
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3127,
          "end": 3137
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3127,
        "end": 3137
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3138,
          "end": 3147
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3138,
        "end": 3147
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 3148,
          "end": 3159
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3148,
        "end": 3159
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 3160,
          "end": 3166
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
                "start": 3176,
                "end": 3186
              }
            },
            "directives": [],
            "loc": {
              "start": 3173,
              "end": 3186
            }
          }
        ],
        "loc": {
          "start": 3167,
          "end": 3188
        }
      },
      "loc": {
        "start": 3160,
        "end": 3188
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 3189,
          "end": 3194
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
                  "start": 3208,
                  "end": 3220
                }
              },
              "loc": {
                "start": 3208,
                "end": 3220
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
                      "start": 3234,
                      "end": 3250
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3231,
                    "end": 3250
                  }
                }
              ],
              "loc": {
                "start": 3221,
                "end": 3256
              }
            },
            "loc": {
              "start": 3201,
              "end": 3256
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
                  "start": 3268,
                  "end": 3272
                }
              },
              "loc": {
                "start": 3268,
                "end": 3272
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
                      "start": 3286,
                      "end": 3294
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3283,
                    "end": 3294
                  }
                }
              ],
              "loc": {
                "start": 3273,
                "end": 3300
              }
            },
            "loc": {
              "start": 3261,
              "end": 3300
            }
          }
        ],
        "loc": {
          "start": 3195,
          "end": 3302
        }
      },
      "loc": {
        "start": 3189,
        "end": 3302
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 3303,
          "end": 3314
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3303,
        "end": 3314
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 3315,
          "end": 3329
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3315,
        "end": 3329
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3330,
          "end": 3335
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3330,
        "end": 3335
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3336,
          "end": 3345
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3336,
        "end": 3345
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 3346,
          "end": 3350
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
                "start": 3360,
                "end": 3368
              }
            },
            "directives": [],
            "loc": {
              "start": 3357,
              "end": 3368
            }
          }
        ],
        "loc": {
          "start": 3351,
          "end": 3370
        }
      },
      "loc": {
        "start": 3346,
        "end": 3370
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 3371,
          "end": 3385
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3371,
        "end": 3385
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 3386,
          "end": 3391
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3386,
        "end": 3391
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 3392,
          "end": 3395
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
                "start": 3402,
                "end": 3411
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3402,
              "end": 3411
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 3416,
                "end": 3427
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3416,
              "end": 3427
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 3432,
                "end": 3443
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3432,
              "end": 3443
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
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
              "value": "canRead",
              "loc": {
                "start": 3462,
                "end": 3469
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3462,
              "end": 3469
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 3474,
                "end": 3482
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3474,
              "end": 3482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 3487,
                "end": 3499
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3487,
              "end": 3499
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 3504,
                "end": 3512
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3504,
              "end": 3512
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 3517,
                "end": 3525
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3517,
              "end": 3525
            }
          }
        ],
        "loc": {
          "start": 3396,
          "end": 3527
        }
      },
      "loc": {
        "start": 3392,
        "end": 3527
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3564,
          "end": 3566
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3564,
        "end": 3566
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3567,
          "end": 3576
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3567,
        "end": 3576
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 3616,
          "end": 3628
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
                "start": 3635,
                "end": 3637
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3635,
              "end": 3637
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 3642,
                "end": 3650
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3642,
              "end": 3650
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 3655,
                "end": 3666
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3655,
              "end": 3666
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3671,
                "end": 3675
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3671,
              "end": 3675
            }
          }
        ],
        "loc": {
          "start": 3629,
          "end": 3677
        }
      },
      "loc": {
        "start": 3616,
        "end": 3677
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 3678,
          "end": 3680
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3678,
        "end": 3680
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 3681,
          "end": 3691
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3681,
        "end": 3691
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 3692,
          "end": 3702
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3692,
        "end": 3702
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "createdBy",
        "loc": {
          "start": 3703,
          "end": 3712
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
                "start": 3719,
                "end": 3721
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3719,
              "end": 3721
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
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
              "value": "handle",
              "loc": {
                "start": 3742,
                "end": 3748
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3742,
              "end": 3748
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 3753,
                "end": 3758
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3753,
              "end": 3758
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 3763,
                "end": 3767
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3763,
              "end": 3767
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 3772,
                "end": 3784
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3772,
              "end": 3784
            }
          }
        ],
        "loc": {
          "start": 3713,
          "end": 3786
        }
      },
      "loc": {
        "start": 3703,
        "end": 3786
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "hasAcceptedAnswer",
        "loc": {
          "start": 3787,
          "end": 3804
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3787,
        "end": 3804
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 3805,
          "end": 3814
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3805,
        "end": 3814
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 3815,
          "end": 3820
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3815,
        "end": 3820
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 3821,
          "end": 3830
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3821,
        "end": 3830
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "answersCount",
        "loc": {
          "start": 3831,
          "end": 3843
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3831,
        "end": 3843
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "commentsCount",
        "loc": {
          "start": 3844,
          "end": 3857
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3844,
        "end": 3857
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsCount",
        "loc": {
          "start": 3858,
          "end": 3870
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 3858,
        "end": 3870
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "forObject",
        "loc": {
          "start": 3871,
          "end": 3880
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
                  "start": 3894,
                  "end": 3897
                }
              },
              "loc": {
                "start": 3894,
                "end": 3897
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
                      "start": 3911,
                      "end": 3918
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3908,
                    "end": 3918
                  }
                }
              ],
              "loc": {
                "start": 3898,
                "end": 3924
              }
            },
            "loc": {
              "start": 3887,
              "end": 3924
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
                  "start": 3936,
                  "end": 3940
                }
              },
              "loc": {
                "start": 3936,
                "end": 3940
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
                      "start": 3954,
                      "end": 3962
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3951,
                    "end": 3962
                  }
                }
              ],
              "loc": {
                "start": 3941,
                "end": 3968
              }
            },
            "loc": {
              "start": 3929,
              "end": 3968
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
                  "start": 3980,
                  "end": 3992
                }
              },
              "loc": {
                "start": 3980,
                "end": 3992
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
                      "start": 4006,
                      "end": 4022
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4003,
                    "end": 4022
                  }
                }
              ],
              "loc": {
                "start": 3993,
                "end": 4028
              }
            },
            "loc": {
              "start": 3973,
              "end": 4028
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
                  "start": 4040,
                  "end": 4047
                }
              },
              "loc": {
                "start": 4040,
                "end": 4047
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
                      "start": 4061,
                      "end": 4072
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4058,
                    "end": 4072
                  }
                }
              ],
              "loc": {
                "start": 4048,
                "end": 4078
              }
            },
            "loc": {
              "start": 4033,
              "end": 4078
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
                  "start": 4090,
                  "end": 4097
                }
              },
              "loc": {
                "start": 4090,
                "end": 4097
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
                      "start": 4111,
                      "end": 4122
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4108,
                    "end": 4122
                  }
                }
              ],
              "loc": {
                "start": 4098,
                "end": 4128
              }
            },
            "loc": {
              "start": 4083,
              "end": 4128
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
                  "start": 4140,
                  "end": 4153
                }
              },
              "loc": {
                "start": 4140,
                "end": 4153
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
                      "start": 4167,
                      "end": 4184
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4164,
                    "end": 4184
                  }
                }
              ],
              "loc": {
                "start": 4154,
                "end": 4190
              }
            },
            "loc": {
              "start": 4133,
              "end": 4190
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
                  "start": 4202,
                  "end": 4210
                }
              },
              "loc": {
                "start": 4202,
                "end": 4210
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
                      "start": 4224,
                      "end": 4236
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4221,
                    "end": 4236
                  }
                }
              ],
              "loc": {
                "start": 4211,
                "end": 4242
              }
            },
            "loc": {
              "start": 4195,
              "end": 4242
            }
          }
        ],
        "loc": {
          "start": 3881,
          "end": 4244
        }
      },
      "loc": {
        "start": 3871,
        "end": 4244
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 4245,
          "end": 4249
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
                "start": 4259,
                "end": 4267
              }
            },
            "directives": [],
            "loc": {
              "start": 4256,
              "end": 4267
            }
          }
        ],
        "loc": {
          "start": 4250,
          "end": 4269
        }
      },
      "loc": {
        "start": 4245,
        "end": 4269
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 4270,
          "end": 4273
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
                "start": 4280,
                "end": 4288
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4280,
              "end": 4288
            }
          }
        ],
        "loc": {
          "start": 4274,
          "end": 4290
        }
      },
      "loc": {
        "start": 4270,
        "end": 4290
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 4328,
          "end": 4336
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
                "start": 4343,
                "end": 4355
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
                      "start": 4366,
                      "end": 4368
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4366,
                    "end": 4368
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 4377,
                      "end": 4385
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4377,
                    "end": 4385
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 4394,
                      "end": 4405
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4394,
                    "end": 4405
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "instructions",
                    "loc": {
                      "start": 4414,
                      "end": 4426
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4414,
                    "end": 4426
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4435,
                      "end": 4439
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4435,
                    "end": 4439
                  }
                }
              ],
              "loc": {
                "start": 4356,
                "end": 4445
              }
            },
            "loc": {
              "start": 4343,
              "end": 4445
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4450,
                "end": 4452
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4450,
              "end": 4452
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4457,
                "end": 4467
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4457,
              "end": 4467
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4472,
                "end": 4482
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4472,
              "end": 4482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "completedAt",
              "loc": {
                "start": 4487,
                "end": 4498
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4487,
              "end": 4498
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isAutomatable",
              "loc": {
                "start": 4503,
                "end": 4516
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4503,
              "end": 4516
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 4521,
                "end": 4531
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4521,
              "end": 4531
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 4536,
                "end": 4545
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4536,
              "end": 4545
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 4550,
                "end": 4558
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4550,
              "end": 4558
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4563,
                "end": 4572
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4563,
              "end": 4572
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "simplicity",
              "loc": {
                "start": 4577,
                "end": 4587
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4577,
              "end": 4587
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesStarted",
              "loc": {
                "start": 4592,
                "end": 4604
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4592,
              "end": 4604
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timesCompleted",
              "loc": {
                "start": 4609,
                "end": 4623
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4609,
              "end": 4623
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractCallData",
              "loc": {
                "start": 4628,
                "end": 4649
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4628,
              "end": 4649
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apiCallData",
              "loc": {
                "start": 4654,
                "end": 4665
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4654,
              "end": 4665
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 4670,
                "end": 4682
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4670,
              "end": 4682
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 4687,
                "end": 4699
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4687,
              "end": 4699
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 4704,
                "end": 4717
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4704,
              "end": 4717
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 4722,
                "end": 4744
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4722,
              "end": 4744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
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
              "value": "inputsCount",
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
              "value": "nodesCount",
              "loc": {
                "start": 4780,
                "end": 4790
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4780,
              "end": 4790
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "nodeLinksCount",
              "loc": {
                "start": 4795,
                "end": 4809
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4795,
              "end": 4809
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "outputsCount",
              "loc": {
                "start": 4814,
                "end": 4826
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4814,
              "end": 4826
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 4831,
                "end": 4843
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4831,
              "end": 4843
            }
          }
        ],
        "loc": {
          "start": 4337,
          "end": 4845
        }
      },
      "loc": {
        "start": 4328,
        "end": 4845
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 4846,
          "end": 4848
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4846,
        "end": 4848
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 4849,
          "end": 4859
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4849,
        "end": 4859
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 4860,
          "end": 4870
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4860,
        "end": 4870
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 4871,
          "end": 4881
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4871,
        "end": 4881
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 4882,
          "end": 4891
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4882,
        "end": 4891
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 4892,
          "end": 4903
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 4892,
        "end": 4903
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 4904,
          "end": 4910
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
                "start": 4920,
                "end": 4930
              }
            },
            "directives": [],
            "loc": {
              "start": 4917,
              "end": 4930
            }
          }
        ],
        "loc": {
          "start": 4911,
          "end": 4932
        }
      },
      "loc": {
        "start": 4904,
        "end": 4932
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 4933,
          "end": 4938
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
                  "start": 4952,
                  "end": 4964
                }
              },
              "loc": {
                "start": 4952,
                "end": 4964
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
                      "start": 4978,
                      "end": 4994
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4975,
                    "end": 4994
                  }
                }
              ],
              "loc": {
                "start": 4965,
                "end": 5000
              }
            },
            "loc": {
              "start": 4945,
              "end": 5000
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
                  "start": 5012,
                  "end": 5016
                }
              },
              "loc": {
                "start": 5012,
                "end": 5016
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
                      "start": 5030,
                      "end": 5038
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5027,
                    "end": 5038
                  }
                }
              ],
              "loc": {
                "start": 5017,
                "end": 5044
              }
            },
            "loc": {
              "start": 5005,
              "end": 5044
            }
          }
        ],
        "loc": {
          "start": 4939,
          "end": 5046
        }
      },
      "loc": {
        "start": 4933,
        "end": 5046
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 5047,
          "end": 5058
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5047,
        "end": 5058
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 5059,
          "end": 5073
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5059,
        "end": 5073
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 5074,
          "end": 5079
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5074,
        "end": 5079
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 5080,
          "end": 5089
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5080,
        "end": 5089
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 5090,
          "end": 5094
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
                "start": 5104,
                "end": 5112
              }
            },
            "directives": [],
            "loc": {
              "start": 5101,
              "end": 5112
            }
          }
        ],
        "loc": {
          "start": 5095,
          "end": 5114
        }
      },
      "loc": {
        "start": 5090,
        "end": 5114
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 5115,
          "end": 5129
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5115,
        "end": 5129
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 5130,
          "end": 5135
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5130,
        "end": 5135
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 5136,
          "end": 5139
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
                "start": 5146,
                "end": 5156
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5146,
              "end": 5156
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canDelete",
              "loc": {
                "start": 5161,
                "end": 5170
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5161,
              "end": 5170
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 5175,
                "end": 5186
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5175,
              "end": 5186
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 5191,
                "end": 5200
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5191,
              "end": 5200
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 5205,
                "end": 5212
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5205,
              "end": 5212
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 5217,
                "end": 5225
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5217,
              "end": 5225
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 5230,
                "end": 5242
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5230,
              "end": 5242
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 5247,
                "end": 5255
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5247,
              "end": 5255
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 5260,
                "end": 5268
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5260,
              "end": 5268
            }
          }
        ],
        "loc": {
          "start": 5140,
          "end": 5270
        }
      },
      "loc": {
        "start": 5136,
        "end": 5270
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5307,
          "end": 5309
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5307,
        "end": 5309
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isInternal",
        "loc": {
          "start": 5310,
          "end": 5320
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5310,
        "end": 5320
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5321,
          "end": 5330
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5321,
        "end": 5330
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 5380,
          "end": 5388
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
                "start": 5395,
                "end": 5407
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
                      "start": 5418,
                      "end": 5420
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5418,
                    "end": 5420
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 5429,
                      "end": 5437
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5429,
                    "end": 5437
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 5446,
                      "end": 5457
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5446,
                    "end": 5457
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 5466,
                      "end": 5478
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5466,
                    "end": 5478
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5487,
                      "end": 5491
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5487,
                    "end": 5491
                  }
                }
              ],
              "loc": {
                "start": 5408,
                "end": 5497
              }
            },
            "loc": {
              "start": 5395,
              "end": 5497
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5502,
                "end": 5504
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5502,
              "end": 5504
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5509,
                "end": 5519
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5509,
              "end": 5519
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5524,
                "end": 5534
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5524,
              "end": 5534
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 5539,
                "end": 5549
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5539,
              "end": 5549
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isDeleted",
              "loc": {
                "start": 5554,
                "end": 5563
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5554,
              "end": 5563
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 5568,
                "end": 5576
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5568,
              "end": 5576
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5581,
                "end": 5590
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5581,
              "end": 5590
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 5595,
                "end": 5602
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5595,
              "end": 5602
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "contractType",
              "loc": {
                "start": 5607,
                "end": 5619
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5607,
              "end": 5619
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "content",
              "loc": {
                "start": 5624,
                "end": 5631
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5624,
              "end": 5631
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 5636,
                "end": 5648
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5636,
              "end": 5648
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 5653,
                "end": 5665
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5653,
              "end": 5665
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 5670,
                "end": 5683
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5670,
              "end": 5683
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 5688,
                "end": 5710
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5688,
              "end": 5710
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 5715,
                "end": 5725
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5715,
              "end": 5725
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 5730,
                "end": 5742
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5730,
              "end": 5742
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5747,
                "end": 5750
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
                      "start": 5761,
                      "end": 5771
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5761,
                    "end": 5771
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 5780,
                      "end": 5787
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5780,
                    "end": 5787
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5796,
                      "end": 5805
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5796,
                    "end": 5805
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 5814,
                      "end": 5823
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5814,
                    "end": 5823
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5832,
                      "end": 5841
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5832,
                    "end": 5841
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 5850,
                      "end": 5856
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5850,
                    "end": 5856
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5865,
                      "end": 5872
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5865,
                    "end": 5872
                  }
                }
              ],
              "loc": {
                "start": 5751,
                "end": 5878
              }
            },
            "loc": {
              "start": 5747,
              "end": 5878
            }
          }
        ],
        "loc": {
          "start": 5389,
          "end": 5880
        }
      },
      "loc": {
        "start": 5380,
        "end": 5880
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5881,
          "end": 5883
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5881,
        "end": 5883
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 5884,
          "end": 5894
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5884,
        "end": 5894
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 5895,
          "end": 5905
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5895,
        "end": 5905
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 5906,
          "end": 5915
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5906,
        "end": 5915
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 5916,
          "end": 5927
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5916,
        "end": 5927
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 5928,
          "end": 5934
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
                "start": 5944,
                "end": 5954
              }
            },
            "directives": [],
            "loc": {
              "start": 5941,
              "end": 5954
            }
          }
        ],
        "loc": {
          "start": 5935,
          "end": 5956
        }
      },
      "loc": {
        "start": 5928,
        "end": 5956
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 5957,
          "end": 5962
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
                  "start": 5976,
                  "end": 5988
                }
              },
              "loc": {
                "start": 5976,
                "end": 5988
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
                      "start": 6002,
                      "end": 6018
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5999,
                    "end": 6018
                  }
                }
              ],
              "loc": {
                "start": 5989,
                "end": 6024
              }
            },
            "loc": {
              "start": 5969,
              "end": 6024
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
                  "start": 6036,
                  "end": 6040
                }
              },
              "loc": {
                "start": 6036,
                "end": 6040
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
                      "start": 6054,
                      "end": 6062
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6051,
                    "end": 6062
                  }
                }
              ],
              "loc": {
                "start": 6041,
                "end": 6068
              }
            },
            "loc": {
              "start": 6029,
              "end": 6068
            }
          }
        ],
        "loc": {
          "start": 5963,
          "end": 6070
        }
      },
      "loc": {
        "start": 5957,
        "end": 6070
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 6071,
          "end": 6082
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6071,
        "end": 6082
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 6083,
          "end": 6097
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6083,
        "end": 6097
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 6098,
          "end": 6103
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6098,
        "end": 6103
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 6104,
          "end": 6113
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6104,
        "end": 6113
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 6114,
          "end": 6118
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
                "start": 6128,
                "end": 6136
              }
            },
            "directives": [],
            "loc": {
              "start": 6125,
              "end": 6136
            }
          }
        ],
        "loc": {
          "start": 6119,
          "end": 6138
        }
      },
      "loc": {
        "start": 6114,
        "end": 6138
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 6139,
          "end": 6153
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6139,
        "end": 6153
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 6154,
          "end": 6159
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6154,
        "end": 6159
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 6160,
          "end": 6163
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
                "start": 6170,
                "end": 6179
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6170,
              "end": 6179
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 6184,
                "end": 6195
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6184,
              "end": 6195
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 6200,
                "end": 6211
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6200,
              "end": 6211
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 6216,
                "end": 6225
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6216,
              "end": 6225
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 6230,
                "end": 6237
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6230,
              "end": 6237
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 6242,
                "end": 6250
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6242,
              "end": 6250
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 6255,
                "end": 6267
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6255,
              "end": 6267
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 6272,
                "end": 6280
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6272,
              "end": 6280
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 6285,
                "end": 6293
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6285,
              "end": 6293
            }
          }
        ],
        "loc": {
          "start": 6164,
          "end": 6295
        }
      },
      "loc": {
        "start": 6160,
        "end": 6295
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6344,
          "end": 6346
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6344,
        "end": 6346
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6347,
          "end": 6356
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6347,
        "end": 6356
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "versions",
        "loc": {
          "start": 6396,
          "end": 6404
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
                "start": 6411,
                "end": 6423
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
                      "start": 6434,
                      "end": 6436
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6434,
                    "end": 6436
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 6445,
                      "end": 6453
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6445,
                    "end": 6453
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 6462,
                      "end": 6473
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6462,
                    "end": 6473
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "jsonVariable",
                    "loc": {
                      "start": 6482,
                      "end": 6494
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6482,
                    "end": 6494
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 6503,
                      "end": 6507
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6503,
                    "end": 6507
                  }
                }
              ],
              "loc": {
                "start": 6424,
                "end": 6513
              }
            },
            "loc": {
              "start": 6411,
              "end": 6513
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6518,
                "end": 6520
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6518,
              "end": 6520
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6525,
                "end": 6535
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6525,
              "end": 6535
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6540,
                "end": 6550
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6540,
              "end": 6550
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isComplete",
              "loc": {
                "start": 6555,
                "end": 6565
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6555,
              "end": 6565
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isFile",
              "loc": {
                "start": 6570,
                "end": 6576
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6570,
              "end": 6576
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLatest",
              "loc": {
                "start": 6581,
                "end": 6589
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6581,
              "end": 6589
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6594,
                "end": 6603
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6594,
              "end": 6603
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "default",
              "loc": {
                "start": 6608,
                "end": 6615
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6608,
              "end": 6615
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardType",
              "loc": {
                "start": 6620,
                "end": 6632
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6620,
              "end": 6632
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "props",
              "loc": {
                "start": 6637,
                "end": 6642
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6637,
              "end": 6642
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "yup",
              "loc": {
                "start": 6647,
                "end": 6650
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6647,
              "end": 6650
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionIndex",
              "loc": {
                "start": 6655,
                "end": 6667
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6655,
              "end": 6667
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "versionLabel",
              "loc": {
                "start": 6672,
                "end": 6684
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6672,
              "end": 6684
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 6689,
                "end": 6702
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6689,
              "end": 6702
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "directoryListingsCount",
              "loc": {
                "start": 6707,
                "end": 6729
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6707,
              "end": 6729
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forksCount",
              "loc": {
                "start": 6734,
                "end": 6744
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6734,
              "end": 6744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 6749,
                "end": 6761
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6749,
              "end": 6761
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6766,
                "end": 6769
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
                      "start": 6780,
                      "end": 6790
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6780,
                    "end": 6790
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canCopy",
                    "loc": {
                      "start": 6799,
                      "end": 6806
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6799,
                    "end": 6806
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 6815,
                      "end": 6824
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6815,
                    "end": 6824
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 6833,
                      "end": 6842
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6833,
                    "end": 6842
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6851,
                      "end": 6860
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6851,
                    "end": 6860
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUse",
                    "loc": {
                      "start": 6869,
                      "end": 6875
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6869,
                    "end": 6875
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6884,
                      "end": 6891
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6884,
                    "end": 6891
                  }
                }
              ],
              "loc": {
                "start": 6770,
                "end": 6897
              }
            },
            "loc": {
              "start": 6766,
              "end": 6897
            }
          }
        ],
        "loc": {
          "start": 6405,
          "end": 6899
        }
      },
      "loc": {
        "start": 6396,
        "end": 6899
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 6900,
          "end": 6902
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6900,
        "end": 6902
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 6903,
          "end": 6913
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6903,
        "end": 6913
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 6914,
          "end": 6924
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6914,
        "end": 6924
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 6925,
          "end": 6934
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6925,
        "end": 6934
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "issuesCount",
        "loc": {
          "start": 6935,
          "end": 6946
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 6935,
        "end": 6946
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "labels",
        "loc": {
          "start": 6947,
          "end": 6953
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
                "start": 6963,
                "end": 6973
              }
            },
            "directives": [],
            "loc": {
              "start": 6960,
              "end": 6973
            }
          }
        ],
        "loc": {
          "start": 6954,
          "end": 6975
        }
      },
      "loc": {
        "start": 6947,
        "end": 6975
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "owner",
        "loc": {
          "start": 6976,
          "end": 6981
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
                  "start": 6995,
                  "end": 7007
                }
              },
              "loc": {
                "start": 6995,
                "end": 7007
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
                      "start": 7021,
                      "end": 7037
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7018,
                    "end": 7037
                  }
                }
              ],
              "loc": {
                "start": 7008,
                "end": 7043
              }
            },
            "loc": {
              "start": 6988,
              "end": 7043
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
                  "start": 7055,
                  "end": 7059
                }
              },
              "loc": {
                "start": 7055,
                "end": 7059
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
                      "start": 7073,
                      "end": 7081
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7070,
                    "end": 7081
                  }
                }
              ],
              "loc": {
                "start": 7060,
                "end": 7087
              }
            },
            "loc": {
              "start": 7048,
              "end": 7087
            }
          }
        ],
        "loc": {
          "start": 6982,
          "end": 7089
        }
      },
      "loc": {
        "start": 6976,
        "end": 7089
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "permissions",
        "loc": {
          "start": 7090,
          "end": 7101
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7090,
        "end": 7101
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "questionsCount",
        "loc": {
          "start": 7102,
          "end": 7116
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7102,
        "end": 7116
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "score",
        "loc": {
          "start": 7117,
          "end": 7122
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7117,
        "end": 7122
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7123,
          "end": 7132
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7123,
        "end": 7132
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tags",
        "loc": {
          "start": 7133,
          "end": 7137
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
                "start": 7147,
                "end": 7155
              }
            },
            "directives": [],
            "loc": {
              "start": 7144,
              "end": 7155
            }
          }
        ],
        "loc": {
          "start": 7138,
          "end": 7157
        }
      },
      "loc": {
        "start": 7133,
        "end": 7157
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "transfersCount",
        "loc": {
          "start": 7158,
          "end": 7172
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7158,
        "end": 7172
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "views",
        "loc": {
          "start": 7173,
          "end": 7178
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7173,
        "end": 7178
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7179,
          "end": 7182
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
                "start": 7189,
                "end": 7198
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7189,
              "end": 7198
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canBookmark",
              "loc": {
                "start": 7203,
                "end": 7214
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7203,
              "end": 7214
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canTransfer",
              "loc": {
                "start": 7219,
                "end": 7230
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7219,
              "end": 7230
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7235,
                "end": 7244
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7235,
              "end": 7244
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canRead",
              "loc": {
                "start": 7249,
                "end": 7256
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7249,
              "end": 7256
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReact",
              "loc": {
                "start": 7261,
                "end": 7269
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7261,
              "end": 7269
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7274,
                "end": 7286
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7274,
              "end": 7286
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7291,
                "end": 7299
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7291,
              "end": 7299
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reaction",
              "loc": {
                "start": 7304,
                "end": 7312
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7304,
              "end": 7312
            }
          }
        ],
        "loc": {
          "start": 7183,
          "end": 7314
        }
      },
      "loc": {
        "start": 7179,
        "end": 7314
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7353,
          "end": 7355
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7353,
        "end": 7355
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isPrivate",
        "loc": {
          "start": 7356,
          "end": 7365
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7356,
        "end": 7365
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7395,
          "end": 7397
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7395,
        "end": 7397
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7398,
          "end": 7408
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7398,
        "end": 7408
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "tag",
        "loc": {
          "start": 7409,
          "end": 7412
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7409,
        "end": 7412
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7413,
          "end": 7422
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7413,
        "end": 7422
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 7423,
          "end": 7435
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
                "start": 7442,
                "end": 7444
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7442,
              "end": 7444
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
              "loc": {
                "start": 7449,
                "end": 7457
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7449,
              "end": 7457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "description",
              "loc": {
                "start": 7462,
                "end": 7473
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7462,
              "end": 7473
            }
          }
        ],
        "loc": {
          "start": 7436,
          "end": 7475
        }
      },
      "loc": {
        "start": 7423,
        "end": 7475
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7476,
          "end": 7479
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
                "start": 7486,
                "end": 7491
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7486,
              "end": 7491
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7496,
                "end": 7508
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7496,
              "end": 7508
            }
          }
        ],
        "loc": {
          "start": 7480,
          "end": 7510
        }
      },
      "loc": {
        "start": 7476,
        "end": 7510
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "translations",
        "loc": {
          "start": 7542,
          "end": 7554
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
                "start": 7561,
                "end": 7563
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7561,
              "end": 7563
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "language",
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
              "value": "bio",
              "loc": {
                "start": 7581,
                "end": 7584
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7581,
              "end": 7584
            }
          }
        ],
        "loc": {
          "start": 7555,
          "end": 7586
        }
      },
      "loc": {
        "start": 7542,
        "end": 7586
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7587,
          "end": 7589
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7587,
        "end": 7589
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 7590,
          "end": 7600
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7590,
        "end": 7600
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7601,
          "end": 7612
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7601,
        "end": 7612
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7613,
          "end": 7619
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7613,
        "end": 7619
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7620,
          "end": 7625
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7620,
        "end": 7625
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7626,
          "end": 7630
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7626,
        "end": 7630
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7631,
          "end": 7643
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7631,
        "end": 7643
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bookmarks",
        "loc": {
          "start": 7644,
          "end": 7653
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7644,
        "end": 7653
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "reportsReceivedCount",
        "loc": {
          "start": 7654,
          "end": 7674
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7654,
        "end": 7674
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "you",
        "loc": {
          "start": 7675,
          "end": 7678
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
                "start": 7685,
                "end": 7694
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7685,
              "end": 7694
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canReport",
              "loc": {
                "start": 7699,
                "end": 7708
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7699,
              "end": 7708
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "canUpdate",
              "loc": {
                "start": 7713,
                "end": 7722
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7713,
              "end": 7722
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBookmarked",
              "loc": {
                "start": 7727,
                "end": 7739
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7727,
              "end": 7739
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isViewed",
              "loc": {
                "start": 7744,
                "end": 7752
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7744,
              "end": 7752
            }
          }
        ],
        "loc": {
          "start": 7679,
          "end": 7754
        }
      },
      "loc": {
        "start": 7675,
        "end": 7754
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 7785,
          "end": 7787
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7785,
        "end": 7787
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "bannerImage",
        "loc": {
          "start": 7788,
          "end": 7799
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7788,
        "end": 7799
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handle",
        "loc": {
          "start": 7800,
          "end": 7806
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7800,
        "end": 7806
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isBot",
        "loc": {
          "start": 7807,
          "end": 7812
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7807,
        "end": 7812
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 7813,
          "end": 7817
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7813,
        "end": 7817
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "profileImage",
        "loc": {
          "start": 7818,
          "end": 7830
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 7818,
        "end": 7830
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
          "start": 10,
          "end": 18
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 22,
            "end": 25
          }
        },
        "loc": {
          "start": 22,
          "end": 25
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
                "start": 28,
                "end": 36
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
                      "start": 43,
                      "end": 55
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
                            "start": 66,
                            "end": 68
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 66,
                          "end": 68
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 77,
                            "end": 85
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 77,
                          "end": 85
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "details",
                          "loc": {
                            "start": 94,
                            "end": 101
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 94,
                          "end": 101
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 110,
                            "end": 114
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 110,
                          "end": 114
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "summary",
                          "loc": {
                            "start": 123,
                            "end": 130
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 123,
                          "end": 130
                        }
                      }
                    ],
                    "loc": {
                      "start": 56,
                      "end": 136
                    }
                  },
                  "loc": {
                    "start": 43,
                    "end": 136
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 141,
                      "end": 143
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 141,
                    "end": 143
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 148,
                      "end": 158
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 148,
                    "end": 158
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 163,
                      "end": 173
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 163,
                    "end": 173
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "callLink",
                    "loc": {
                      "start": 178,
                      "end": 186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 178,
                    "end": 186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 191,
                      "end": 204
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 191,
                    "end": 204
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "documentationLink",
                    "loc": {
                      "start": 209,
                      "end": 226
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 209,
                    "end": 226
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 231,
                      "end": 241
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 231,
                    "end": 241
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 246,
                      "end": 254
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 246,
                    "end": 254
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
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
                    "value": "reportsCount",
                    "loc": {
                      "start": 273,
                      "end": 285
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 273,
                    "end": 285
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 290,
                      "end": 302
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 290,
                    "end": 302
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 307,
                      "end": 319
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 307,
                    "end": 319
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 324,
                      "end": 327
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
                            "start": 338,
                            "end": 348
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 338,
                          "end": 348
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 357,
                            "end": 364
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 357,
                          "end": 364
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 373,
                            "end": 382
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 373,
                          "end": 382
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 391,
                            "end": 400
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 391,
                          "end": 400
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 409,
                            "end": 418
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 409,
                          "end": 418
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 427,
                            "end": 433
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 427,
                          "end": 433
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 442,
                            "end": 449
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 442,
                          "end": 449
                        }
                      }
                    ],
                    "loc": {
                      "start": 328,
                      "end": 455
                    }
                  },
                  "loc": {
                    "start": 324,
                    "end": 455
                  }
                }
              ],
              "loc": {
                "start": 37,
                "end": 457
              }
            },
            "loc": {
              "start": 28,
              "end": 457
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 458,
                "end": 460
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 458,
              "end": 460
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 461,
                "end": 471
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 461,
              "end": 471
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 472,
                "end": 482
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 472,
              "end": 482
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 483,
                "end": 492
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 483,
              "end": 492
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 493,
                "end": 504
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 493,
              "end": 504
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 505,
                "end": 511
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
                      "start": 521,
                      "end": 531
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 518,
                    "end": 531
                  }
                }
              ],
              "loc": {
                "start": 512,
                "end": 533
              }
            },
            "loc": {
              "start": 505,
              "end": 533
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 534,
                "end": 539
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
                        "start": 553,
                        "end": 565
                      }
                    },
                    "loc": {
                      "start": 553,
                      "end": 565
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
                            "start": 579,
                            "end": 595
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 576,
                          "end": 595
                        }
                      }
                    ],
                    "loc": {
                      "start": 566,
                      "end": 601
                    }
                  },
                  "loc": {
                    "start": 546,
                    "end": 601
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
                        "start": 613,
                        "end": 617
                      }
                    },
                    "loc": {
                      "start": 613,
                      "end": 617
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
                            "start": 631,
                            "end": 639
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 628,
                          "end": 639
                        }
                      }
                    ],
                    "loc": {
                      "start": 618,
                      "end": 645
                    }
                  },
                  "loc": {
                    "start": 606,
                    "end": 645
                  }
                }
              ],
              "loc": {
                "start": 540,
                "end": 647
              }
            },
            "loc": {
              "start": 534,
              "end": 647
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 648,
                "end": 659
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 648,
              "end": 659
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 660,
                "end": 674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 660,
              "end": 674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 675,
                "end": 680
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 675,
              "end": 680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 681,
                "end": 690
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 681,
              "end": 690
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 691,
                "end": 695
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
                      "start": 705,
                      "end": 713
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 702,
                    "end": 713
                  }
                }
              ],
              "loc": {
                "start": 696,
                "end": 715
              }
            },
            "loc": {
              "start": 691,
              "end": 715
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 716,
                "end": 730
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 716,
              "end": 730
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 731,
                "end": 736
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 731,
              "end": 736
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 737,
                "end": 740
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
                      "start": 747,
                      "end": 756
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 747,
                    "end": 756
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 761,
                      "end": 772
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 761,
                    "end": 772
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 777,
                      "end": 788
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 777,
                    "end": 788
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 793,
                      "end": 802
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 793,
                    "end": 802
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 807,
                      "end": 814
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 807,
                    "end": 814
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 819,
                      "end": 827
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 819,
                    "end": 827
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 832,
                      "end": 844
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 832,
                    "end": 844
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 849,
                      "end": 857
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 849,
                    "end": 857
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 862,
                      "end": 870
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 862,
                    "end": 870
                  }
                }
              ],
              "loc": {
                "start": 741,
                "end": 872
              }
            },
            "loc": {
              "start": 737,
              "end": 872
            }
          }
        ],
        "loc": {
          "start": 26,
          "end": 874
        }
      },
      "loc": {
        "start": 1,
        "end": 874
      }
    },
    "Api_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Api_nav",
        "loc": {
          "start": 884,
          "end": 891
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Api",
          "loc": {
            "start": 895,
            "end": 898
          }
        },
        "loc": {
          "start": 895,
          "end": 898
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
                "start": 901,
                "end": 903
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 901,
              "end": 903
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 904,
                "end": 913
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 904,
              "end": 913
            }
          }
        ],
        "loc": {
          "start": 899,
          "end": 915
        }
      },
      "loc": {
        "start": 875,
        "end": 915
      }
    },
    "Label_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Label_list",
        "loc": {
          "start": 925,
          "end": 935
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Label",
          "loc": {
            "start": 939,
            "end": 944
          }
        },
        "loc": {
          "start": 939,
          "end": 944
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
                "start": 947,
                "end": 949
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 947,
              "end": 949
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 950,
                "end": 960
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 950,
              "end": 960
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 961,
                "end": 971
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 961,
              "end": 971
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "color",
              "loc": {
                "start": 972,
                "end": 977
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 972,
              "end": 977
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "label",
              "loc": {
                "start": 978,
                "end": 983
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 978,
              "end": 983
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 984,
                "end": 989
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
                        "start": 1003,
                        "end": 1015
                      }
                    },
                    "loc": {
                      "start": 1003,
                      "end": 1015
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
                            "start": 1029,
                            "end": 1045
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1026,
                          "end": 1045
                        }
                      }
                    ],
                    "loc": {
                      "start": 1016,
                      "end": 1051
                    }
                  },
                  "loc": {
                    "start": 996,
                    "end": 1051
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
                        "start": 1063,
                        "end": 1067
                      }
                    },
                    "loc": {
                      "start": 1063,
                      "end": 1067
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
                            "start": 1081,
                            "end": 1089
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1078,
                          "end": 1089
                        }
                      }
                    ],
                    "loc": {
                      "start": 1068,
                      "end": 1095
                    }
                  },
                  "loc": {
                    "start": 1056,
                    "end": 1095
                  }
                }
              ],
              "loc": {
                "start": 990,
                "end": 1097
              }
            },
            "loc": {
              "start": 984,
              "end": 1097
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1098,
                "end": 1101
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
                      "start": 1108,
                      "end": 1117
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1108,
                    "end": 1117
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1122,
                      "end": 1131
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1122,
                    "end": 1131
                  }
                }
              ],
              "loc": {
                "start": 1102,
                "end": 1133
              }
            },
            "loc": {
              "start": 1098,
              "end": 1133
            }
          }
        ],
        "loc": {
          "start": 945,
          "end": 1135
        }
      },
      "loc": {
        "start": 916,
        "end": 1135
      }
    },
    "Note_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_list",
        "loc": {
          "start": 1145,
          "end": 1154
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 1158,
            "end": 1162
          }
        },
        "loc": {
          "start": 1158,
          "end": 1162
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
                "start": 1165,
                "end": 1173
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
                      "start": 1180,
                      "end": 1192
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
                            "start": 1203,
                            "end": 1205
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1203,
                          "end": 1205
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 1214,
                            "end": 1222
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1214,
                          "end": 1222
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 1231,
                            "end": 1242
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1231,
                          "end": 1242
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 1251,
                            "end": 1255
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1251,
                          "end": 1255
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "text",
                          "loc": {
                            "start": 1264,
                            "end": 1268
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1264,
                          "end": 1268
                        }
                      }
                    ],
                    "loc": {
                      "start": 1193,
                      "end": 1274
                    }
                  },
                  "loc": {
                    "start": 1180,
                    "end": 1274
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 1279,
                      "end": 1281
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1279,
                    "end": 1281
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 1286,
                      "end": 1296
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1286,
                    "end": 1296
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 1301,
                      "end": 1311
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1301,
                    "end": 1311
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 1316,
                      "end": 1324
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1316,
                    "end": 1324
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 1329,
                      "end": 1338
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1329,
                    "end": 1338
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 1343,
                      "end": 1355
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1343,
                    "end": 1355
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 1360,
                      "end": 1372
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1360,
                    "end": 1372
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 1377,
                      "end": 1389
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1377,
                    "end": 1389
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 1394,
                      "end": 1397
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
                            "start": 1408,
                            "end": 1418
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1408,
                          "end": 1418
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 1427,
                            "end": 1434
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1427,
                          "end": 1434
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 1443,
                            "end": 1452
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1443,
                          "end": 1452
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 1461,
                            "end": 1470
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1461,
                          "end": 1470
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 1479,
                            "end": 1488
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1479,
                          "end": 1488
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 1497,
                            "end": 1503
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1497,
                          "end": 1503
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 1512,
                            "end": 1519
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 1512,
                          "end": 1519
                        }
                      }
                    ],
                    "loc": {
                      "start": 1398,
                      "end": 1525
                    }
                  },
                  "loc": {
                    "start": 1394,
                    "end": 1525
                  }
                }
              ],
              "loc": {
                "start": 1174,
                "end": 1527
              }
            },
            "loc": {
              "start": 1165,
              "end": 1527
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 1528,
                "end": 1530
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1528,
              "end": 1530
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 1531,
                "end": 1541
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1531,
              "end": 1541
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 1542,
                "end": 1552
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1542,
              "end": 1552
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1553,
                "end": 1562
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1553,
              "end": 1562
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 1563,
                "end": 1574
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1563,
              "end": 1574
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 1575,
                "end": 1581
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
                      "start": 1591,
                      "end": 1601
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1588,
                    "end": 1601
                  }
                }
              ],
              "loc": {
                "start": 1582,
                "end": 1603
              }
            },
            "loc": {
              "start": 1575,
              "end": 1603
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 1604,
                "end": 1609
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
                        "start": 1623,
                        "end": 1635
                      }
                    },
                    "loc": {
                      "start": 1623,
                      "end": 1635
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
                            "start": 1649,
                            "end": 1665
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1646,
                          "end": 1665
                        }
                      }
                    ],
                    "loc": {
                      "start": 1636,
                      "end": 1671
                    }
                  },
                  "loc": {
                    "start": 1616,
                    "end": 1671
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
                        "start": 1683,
                        "end": 1687
                      }
                    },
                    "loc": {
                      "start": 1683,
                      "end": 1687
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
                            "start": 1701,
                            "end": 1709
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 1698,
                          "end": 1709
                        }
                      }
                    ],
                    "loc": {
                      "start": 1688,
                      "end": 1715
                    }
                  },
                  "loc": {
                    "start": 1676,
                    "end": 1715
                  }
                }
              ],
              "loc": {
                "start": 1610,
                "end": 1717
              }
            },
            "loc": {
              "start": 1604,
              "end": 1717
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 1718,
                "end": 1729
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1718,
              "end": 1729
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 1730,
                "end": 1744
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1730,
              "end": 1744
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 1745,
                "end": 1750
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1745,
              "end": 1750
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 1751,
                "end": 1760
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1751,
              "end": 1760
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 1761,
                "end": 1765
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
                      "start": 1775,
                      "end": 1783
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 1772,
                    "end": 1783
                  }
                }
              ],
              "loc": {
                "start": 1766,
                "end": 1785
              }
            },
            "loc": {
              "start": 1761,
              "end": 1785
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 1786,
                "end": 1800
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1786,
              "end": 1800
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 1801,
                "end": 1806
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1801,
              "end": 1806
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 1807,
                "end": 1810
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
                      "start": 1817,
                      "end": 1826
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1817,
                    "end": 1826
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 1831,
                      "end": 1842
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1831,
                    "end": 1842
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 1847,
                      "end": 1858
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1847,
                    "end": 1858
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 1863,
                      "end": 1872
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1863,
                    "end": 1872
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 1877,
                      "end": 1884
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1877,
                    "end": 1884
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 1889,
                      "end": 1897
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1889,
                    "end": 1897
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 1902,
                      "end": 1914
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1902,
                    "end": 1914
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 1919,
                      "end": 1927
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1919,
                    "end": 1927
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 1932,
                      "end": 1940
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 1932,
                    "end": 1940
                  }
                }
              ],
              "loc": {
                "start": 1811,
                "end": 1942
              }
            },
            "loc": {
              "start": 1807,
              "end": 1942
            }
          }
        ],
        "loc": {
          "start": 1163,
          "end": 1944
        }
      },
      "loc": {
        "start": 1136,
        "end": 1944
      }
    },
    "Note_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Note_nav",
        "loc": {
          "start": 1954,
          "end": 1962
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Note",
          "loc": {
            "start": 1966,
            "end": 1970
          }
        },
        "loc": {
          "start": 1966,
          "end": 1970
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
                "start": 1973,
                "end": 1975
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1973,
              "end": 1975
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 1976,
                "end": 1985
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 1976,
              "end": 1985
            }
          }
        ],
        "loc": {
          "start": 1971,
          "end": 1987
        }
      },
      "loc": {
        "start": 1945,
        "end": 1987
      }
    },
    "Organization_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_list",
        "loc": {
          "start": 1997,
          "end": 2014
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 2018,
            "end": 2030
          }
        },
        "loc": {
          "start": 2018,
          "end": 2030
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
                "start": 2033,
                "end": 2035
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2033,
              "end": 2035
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2036,
                "end": 2047
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2036,
              "end": 2047
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2048,
                "end": 2054
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2048,
              "end": 2054
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 2055,
                "end": 2065
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2055,
              "end": 2065
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 2066,
                "end": 2076
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2066,
              "end": 2076
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isOpenToNewMembers",
              "loc": {
                "start": 2077,
                "end": 2095
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2077,
              "end": 2095
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 2096,
                "end": 2105
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2096,
              "end": 2105
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 2106,
                "end": 2119
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2106,
              "end": 2119
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membersCount",
              "loc": {
                "start": 2120,
                "end": 2132
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2120,
              "end": 2132
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2133,
                "end": 2145
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2133,
              "end": 2145
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 2146,
                "end": 2158
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2146,
              "end": 2158
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 2159,
                "end": 2168
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2159,
              "end": 2168
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 2169,
                "end": 2173
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
                      "start": 2183,
                      "end": 2191
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 2180,
                    "end": 2191
                  }
                }
              ],
              "loc": {
                "start": 2174,
                "end": 2193
              }
            },
            "loc": {
              "start": 2169,
              "end": 2193
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 2194,
                "end": 2206
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
                      "start": 2213,
                      "end": 2215
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2213,
                    "end": 2215
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 2220,
                      "end": 2228
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2220,
                    "end": 2228
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bio",
                    "loc": {
                      "start": 2233,
                      "end": 2236
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2233,
                    "end": 2236
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 2241,
                      "end": 2245
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2241,
                    "end": 2245
                  }
                }
              ],
              "loc": {
                "start": 2207,
                "end": 2247
              }
            },
            "loc": {
              "start": 2194,
              "end": 2247
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2248,
                "end": 2251
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
                      "start": 2258,
                      "end": 2271
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2258,
                    "end": 2271
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2276,
                      "end": 2285
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2276,
                    "end": 2285
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2290,
                      "end": 2301
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2290,
                    "end": 2301
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2306,
                      "end": 2315
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2306,
                    "end": 2315
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2320,
                      "end": 2329
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2320,
                    "end": 2329
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2334,
                      "end": 2341
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2334,
                    "end": 2341
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2346,
                      "end": 2358
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2346,
                    "end": 2358
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2363,
                      "end": 2371
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2363,
                    "end": 2371
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 2376,
                      "end": 2390
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
                            "start": 2401,
                            "end": 2403
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2401,
                          "end": 2403
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2412,
                            "end": 2422
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2412,
                          "end": 2422
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2431,
                            "end": 2441
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2431,
                          "end": 2441
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 2450,
                            "end": 2457
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2450,
                          "end": 2457
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 2466,
                            "end": 2477
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2466,
                          "end": 2477
                        }
                      }
                    ],
                    "loc": {
                      "start": 2391,
                      "end": 2483
                    }
                  },
                  "loc": {
                    "start": 2376,
                    "end": 2483
                  }
                }
              ],
              "loc": {
                "start": 2252,
                "end": 2485
              }
            },
            "loc": {
              "start": 2248,
              "end": 2485
            }
          }
        ],
        "loc": {
          "start": 2031,
          "end": 2487
        }
      },
      "loc": {
        "start": 1988,
        "end": 2487
      }
    },
    "Organization_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Organization_nav",
        "loc": {
          "start": 2497,
          "end": 2513
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Organization",
          "loc": {
            "start": 2517,
            "end": 2529
          }
        },
        "loc": {
          "start": 2517,
          "end": 2529
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
                "start": 2532,
                "end": 2534
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2532,
              "end": 2534
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 2535,
                "end": 2546
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2535,
              "end": 2546
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 2547,
                "end": 2553
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2547,
              "end": 2553
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 2554,
                "end": 2566
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2554,
              "end": 2566
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 2567,
                "end": 2570
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
                      "start": 2577,
                      "end": 2590
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2577,
                    "end": 2590
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 2595,
                      "end": 2604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2595,
                    "end": 2604
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 2609,
                      "end": 2620
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2609,
                    "end": 2620
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 2625,
                      "end": 2634
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2625,
                    "end": 2634
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 2639,
                      "end": 2648
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2639,
                    "end": 2648
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 2653,
                      "end": 2660
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2653,
                    "end": 2660
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 2665,
                      "end": 2677
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2665,
                    "end": 2677
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 2682,
                      "end": 2690
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2682,
                    "end": 2690
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yourMembership",
                    "loc": {
                      "start": 2695,
                      "end": 2709
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
                            "start": 2720,
                            "end": 2722
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2720,
                          "end": 2722
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2731,
                            "end": 2741
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2731,
                          "end": 2741
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2750,
                            "end": 2760
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2750,
                          "end": 2760
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "isAdmin",
                          "loc": {
                            "start": 2769,
                            "end": 2776
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2769,
                          "end": 2776
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "permissions",
                          "loc": {
                            "start": 2785,
                            "end": 2796
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2785,
                          "end": 2796
                        }
                      }
                    ],
                    "loc": {
                      "start": 2710,
                      "end": 2802
                    }
                  },
                  "loc": {
                    "start": 2695,
                    "end": 2802
                  }
                }
              ],
              "loc": {
                "start": 2571,
                "end": 2804
              }
            },
            "loc": {
              "start": 2567,
              "end": 2804
            }
          }
        ],
        "loc": {
          "start": 2530,
          "end": 2806
        }
      },
      "loc": {
        "start": 2488,
        "end": 2806
      }
    },
    "Project_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_list",
        "loc": {
          "start": 2816,
          "end": 2828
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 2832,
            "end": 2839
          }
        },
        "loc": {
          "start": 2832,
          "end": 2839
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
                "start": 2842,
                "end": 2850
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
                      "start": 2857,
                      "end": 2869
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
                            "start": 2880,
                            "end": 2882
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2880,
                          "end": 2882
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 2891,
                            "end": 2899
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2891,
                          "end": 2899
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2908,
                            "end": 2919
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2908,
                          "end": 2919
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2928,
                            "end": 2932
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2928,
                          "end": 2932
                        }
                      }
                    ],
                    "loc": {
                      "start": 2870,
                      "end": 2938
                    }
                  },
                  "loc": {
                    "start": 2857,
                    "end": 2938
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2943,
                      "end": 2945
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2943,
                    "end": 2945
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2950,
                      "end": 2960
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2950,
                    "end": 2960
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2965,
                      "end": 2975
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2965,
                    "end": 2975
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoriesCount",
                    "loc": {
                      "start": 2980,
                      "end": 2996
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2980,
                    "end": 2996
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 3001,
                      "end": 3009
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3001,
                    "end": 3009
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 3014,
                      "end": 3023
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3014,
                    "end": 3023
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 3028,
                      "end": 3040
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3028,
                    "end": 3040
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "runProjectsCount",
                    "loc": {
                      "start": 3045,
                      "end": 3061
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3045,
                    "end": 3061
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 3066,
                      "end": 3076
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3066,
                    "end": 3076
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 3081,
                      "end": 3093
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3081,
                    "end": 3093
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 3098,
                      "end": 3110
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3098,
                    "end": 3110
                  }
                }
              ],
              "loc": {
                "start": 2851,
                "end": 3112
              }
            },
            "loc": {
              "start": 2842,
              "end": 3112
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3113,
                "end": 3115
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3113,
              "end": 3115
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3116,
                "end": 3126
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3116,
              "end": 3126
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3127,
                "end": 3137
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3127,
              "end": 3137
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3138,
                "end": 3147
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3138,
              "end": 3147
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 3148,
                "end": 3159
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3148,
              "end": 3159
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 3160,
                "end": 3166
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
                      "start": 3176,
                      "end": 3186
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3173,
                    "end": 3186
                  }
                }
              ],
              "loc": {
                "start": 3167,
                "end": 3188
              }
            },
            "loc": {
              "start": 3160,
              "end": 3188
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 3189,
                "end": 3194
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
                        "start": 3208,
                        "end": 3220
                      }
                    },
                    "loc": {
                      "start": 3208,
                      "end": 3220
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
                            "start": 3234,
                            "end": 3250
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3231,
                          "end": 3250
                        }
                      }
                    ],
                    "loc": {
                      "start": 3221,
                      "end": 3256
                    }
                  },
                  "loc": {
                    "start": 3201,
                    "end": 3256
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
                        "start": 3268,
                        "end": 3272
                      }
                    },
                    "loc": {
                      "start": 3268,
                      "end": 3272
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
                            "start": 3286,
                            "end": 3294
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3283,
                          "end": 3294
                        }
                      }
                    ],
                    "loc": {
                      "start": 3273,
                      "end": 3300
                    }
                  },
                  "loc": {
                    "start": 3261,
                    "end": 3300
                  }
                }
              ],
              "loc": {
                "start": 3195,
                "end": 3302
              }
            },
            "loc": {
              "start": 3189,
              "end": 3302
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 3303,
                "end": 3314
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3303,
              "end": 3314
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 3315,
                "end": 3329
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3315,
              "end": 3329
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3330,
                "end": 3335
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3330,
              "end": 3335
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3336,
                "end": 3345
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3336,
              "end": 3345
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 3346,
                "end": 3350
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
                      "start": 3360,
                      "end": 3368
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 3357,
                    "end": 3368
                  }
                }
              ],
              "loc": {
                "start": 3351,
                "end": 3370
              }
            },
            "loc": {
              "start": 3346,
              "end": 3370
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 3371,
                "end": 3385
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3371,
              "end": 3385
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 3386,
                "end": 3391
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3386,
              "end": 3391
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 3392,
                "end": 3395
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
                      "start": 3402,
                      "end": 3411
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3402,
                    "end": 3411
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 3416,
                      "end": 3427
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3416,
                    "end": 3427
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 3432,
                      "end": 3443
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3432,
                    "end": 3443
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
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
                    "value": "canRead",
                    "loc": {
                      "start": 3462,
                      "end": 3469
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3462,
                    "end": 3469
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 3474,
                      "end": 3482
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3474,
                    "end": 3482
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 3487,
                      "end": 3499
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3487,
                    "end": 3499
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 3504,
                      "end": 3512
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3504,
                    "end": 3512
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 3517,
                      "end": 3525
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3517,
                    "end": 3525
                  }
                }
              ],
              "loc": {
                "start": 3396,
                "end": 3527
              }
            },
            "loc": {
              "start": 3392,
              "end": 3527
            }
          }
        ],
        "loc": {
          "start": 2840,
          "end": 3529
        }
      },
      "loc": {
        "start": 2807,
        "end": 3529
      }
    },
    "Project_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Project_nav",
        "loc": {
          "start": 3539,
          "end": 3550
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Project",
          "loc": {
            "start": 3554,
            "end": 3561
          }
        },
        "loc": {
          "start": 3554,
          "end": 3561
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
                "start": 3564,
                "end": 3566
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3564,
              "end": 3566
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3567,
                "end": 3576
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3567,
              "end": 3576
            }
          }
        ],
        "loc": {
          "start": 3562,
          "end": 3578
        }
      },
      "loc": {
        "start": 3530,
        "end": 3578
      }
    },
    "Question_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Question_list",
        "loc": {
          "start": 3588,
          "end": 3601
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Question",
          "loc": {
            "start": 3605,
            "end": 3613
          }
        },
        "loc": {
          "start": 3605,
          "end": 3613
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
                "start": 3616,
                "end": 3628
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
                      "start": 3635,
                      "end": 3637
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3635,
                    "end": 3637
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 3642,
                      "end": 3650
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3642,
                    "end": 3650
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 3655,
                      "end": 3666
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3655,
                    "end": 3666
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3671,
                      "end": 3675
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3671,
                    "end": 3675
                  }
                }
              ],
              "loc": {
                "start": 3629,
                "end": 3677
              }
            },
            "loc": {
              "start": 3616,
              "end": 3677
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 3678,
                "end": 3680
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3678,
              "end": 3680
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 3681,
                "end": 3691
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3681,
              "end": 3691
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 3692,
                "end": 3702
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3692,
              "end": 3702
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "createdBy",
              "loc": {
                "start": 3703,
                "end": 3712
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
                      "start": 3719,
                      "end": 3721
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3719,
                    "end": 3721
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bannerImage",
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
                    "value": "handle",
                    "loc": {
                      "start": 3742,
                      "end": 3748
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3742,
                    "end": 3748
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBot",
                    "loc": {
                      "start": 3753,
                      "end": 3758
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3753,
                    "end": 3758
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 3763,
                      "end": 3767
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3763,
                    "end": 3767
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "profileImage",
                    "loc": {
                      "start": 3772,
                      "end": 3784
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 3772,
                    "end": 3784
                  }
                }
              ],
              "loc": {
                "start": 3713,
                "end": 3786
              }
            },
            "loc": {
              "start": 3703,
              "end": 3786
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasAcceptedAnswer",
              "loc": {
                "start": 3787,
                "end": 3804
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3787,
              "end": 3804
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 3805,
                "end": 3814
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3805,
              "end": 3814
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 3815,
                "end": 3820
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3815,
              "end": 3820
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 3821,
                "end": 3830
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3821,
              "end": 3830
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "answersCount",
              "loc": {
                "start": 3831,
                "end": 3843
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3831,
              "end": 3843
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "commentsCount",
              "loc": {
                "start": 3844,
                "end": 3857
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3844,
              "end": 3857
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsCount",
              "loc": {
                "start": 3858,
                "end": 3870
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 3858,
              "end": 3870
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "forObject",
              "loc": {
                "start": 3871,
                "end": 3880
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
                        "start": 3894,
                        "end": 3897
                      }
                    },
                    "loc": {
                      "start": 3894,
                      "end": 3897
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
                            "start": 3911,
                            "end": 3918
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3908,
                          "end": 3918
                        }
                      }
                    ],
                    "loc": {
                      "start": 3898,
                      "end": 3924
                    }
                  },
                  "loc": {
                    "start": 3887,
                    "end": 3924
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
                        "start": 3936,
                        "end": 3940
                      }
                    },
                    "loc": {
                      "start": 3936,
                      "end": 3940
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
                            "start": 3954,
                            "end": 3962
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 3951,
                          "end": 3962
                        }
                      }
                    ],
                    "loc": {
                      "start": 3941,
                      "end": 3968
                    }
                  },
                  "loc": {
                    "start": 3929,
                    "end": 3968
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
                        "start": 3980,
                        "end": 3992
                      }
                    },
                    "loc": {
                      "start": 3980,
                      "end": 3992
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
                            "start": 4006,
                            "end": 4022
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4003,
                          "end": 4022
                        }
                      }
                    ],
                    "loc": {
                      "start": 3993,
                      "end": 4028
                    }
                  },
                  "loc": {
                    "start": 3973,
                    "end": 4028
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
                        "start": 4040,
                        "end": 4047
                      }
                    },
                    "loc": {
                      "start": 4040,
                      "end": 4047
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
                            "start": 4061,
                            "end": 4072
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4058,
                          "end": 4072
                        }
                      }
                    ],
                    "loc": {
                      "start": 4048,
                      "end": 4078
                    }
                  },
                  "loc": {
                    "start": 4033,
                    "end": 4078
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
                        "start": 4090,
                        "end": 4097
                      }
                    },
                    "loc": {
                      "start": 4090,
                      "end": 4097
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
                            "start": 4111,
                            "end": 4122
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4108,
                          "end": 4122
                        }
                      }
                    ],
                    "loc": {
                      "start": 4098,
                      "end": 4128
                    }
                  },
                  "loc": {
                    "start": 4083,
                    "end": 4128
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
                        "start": 4140,
                        "end": 4153
                      }
                    },
                    "loc": {
                      "start": 4140,
                      "end": 4153
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
                            "start": 4167,
                            "end": 4184
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4164,
                          "end": 4184
                        }
                      }
                    ],
                    "loc": {
                      "start": 4154,
                      "end": 4190
                    }
                  },
                  "loc": {
                    "start": 4133,
                    "end": 4190
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
                        "start": 4202,
                        "end": 4210
                      }
                    },
                    "loc": {
                      "start": 4202,
                      "end": 4210
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
                            "start": 4224,
                            "end": 4236
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4221,
                          "end": 4236
                        }
                      }
                    ],
                    "loc": {
                      "start": 4211,
                      "end": 4242
                    }
                  },
                  "loc": {
                    "start": 4195,
                    "end": 4242
                  }
                }
              ],
              "loc": {
                "start": 3881,
                "end": 4244
              }
            },
            "loc": {
              "start": 3871,
              "end": 4244
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 4245,
                "end": 4249
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
                      "start": 4259,
                      "end": 4267
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4256,
                    "end": 4267
                  }
                }
              ],
              "loc": {
                "start": 4250,
                "end": 4269
              }
            },
            "loc": {
              "start": 4245,
              "end": 4269
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 4270,
                "end": 4273
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
                      "start": 4280,
                      "end": 4288
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4280,
                    "end": 4288
                  }
                }
              ],
              "loc": {
                "start": 4274,
                "end": 4290
              }
            },
            "loc": {
              "start": 4270,
              "end": 4290
            }
          }
        ],
        "loc": {
          "start": 3614,
          "end": 4292
        }
      },
      "loc": {
        "start": 3579,
        "end": 4292
      }
    },
    "Routine_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_list",
        "loc": {
          "start": 4302,
          "end": 4314
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 4318,
            "end": 4325
          }
        },
        "loc": {
          "start": 4318,
          "end": 4325
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
                "start": 4328,
                "end": 4336
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
                      "start": 4343,
                      "end": 4355
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
                            "start": 4366,
                            "end": 4368
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4366,
                          "end": 4368
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 4377,
                            "end": 4385
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4377,
                          "end": 4385
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4394,
                            "end": 4405
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4394,
                          "end": 4405
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "instructions",
                          "loc": {
                            "start": 4414,
                            "end": 4426
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4414,
                          "end": 4426
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4435,
                            "end": 4439
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4435,
                          "end": 4439
                        }
                      }
                    ],
                    "loc": {
                      "start": 4356,
                      "end": 4445
                    }
                  },
                  "loc": {
                    "start": 4343,
                    "end": 4445
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4450,
                      "end": 4452
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4450,
                    "end": 4452
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4457,
                      "end": 4467
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4457,
                    "end": 4467
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4472,
                      "end": 4482
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4472,
                    "end": 4482
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "completedAt",
                    "loc": {
                      "start": 4487,
                      "end": 4498
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4487,
                    "end": 4498
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isAutomatable",
                    "loc": {
                      "start": 4503,
                      "end": 4516
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4503,
                    "end": 4516
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 4521,
                      "end": 4531
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4521,
                    "end": 4531
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 4536,
                      "end": 4545
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4536,
                    "end": 4545
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 4550,
                      "end": 4558
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4550,
                    "end": 4558
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 4563,
                      "end": 4572
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4563,
                    "end": 4572
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "simplicity",
                    "loc": {
                      "start": 4577,
                      "end": 4587
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4577,
                    "end": 4587
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesStarted",
                    "loc": {
                      "start": 4592,
                      "end": 4604
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4592,
                    "end": 4604
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "timesCompleted",
                    "loc": {
                      "start": 4609,
                      "end": 4623
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4609,
                    "end": 4623
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractCallData",
                    "loc": {
                      "start": 4628,
                      "end": 4649
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4628,
                    "end": 4649
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apiCallData",
                    "loc": {
                      "start": 4654,
                      "end": 4665
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4654,
                    "end": 4665
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 4670,
                      "end": 4682
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4670,
                    "end": 4682
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 4687,
                      "end": 4699
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4687,
                    "end": 4699
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 4704,
                      "end": 4717
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4704,
                    "end": 4717
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 4722,
                      "end": 4744
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4722,
                    "end": 4744
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
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
                    "value": "inputsCount",
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
                    "value": "nodesCount",
                    "loc": {
                      "start": 4780,
                      "end": 4790
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4780,
                    "end": 4790
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "nodeLinksCount",
                    "loc": {
                      "start": 4795,
                      "end": 4809
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4795,
                    "end": 4809
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "outputsCount",
                    "loc": {
                      "start": 4814,
                      "end": 4826
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4814,
                    "end": 4826
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 4831,
                      "end": 4843
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4831,
                    "end": 4843
                  }
                }
              ],
              "loc": {
                "start": 4337,
                "end": 4845
              }
            },
            "loc": {
              "start": 4328,
              "end": 4845
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4846,
                "end": 4848
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4846,
              "end": 4848
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 4849,
                "end": 4859
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4849,
              "end": 4859
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 4860,
                "end": 4870
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4860,
              "end": 4870
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 4871,
                "end": 4881
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4871,
              "end": 4881
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 4882,
                "end": 4891
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4882,
              "end": 4891
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 4892,
                "end": 4903
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4892,
              "end": 4903
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 4904,
                "end": 4910
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
                      "start": 4920,
                      "end": 4930
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 4917,
                    "end": 4930
                  }
                }
              ],
              "loc": {
                "start": 4911,
                "end": 4932
              }
            },
            "loc": {
              "start": 4904,
              "end": 4932
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 4933,
                "end": 4938
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
                        "start": 4952,
                        "end": 4964
                      }
                    },
                    "loc": {
                      "start": 4952,
                      "end": 4964
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
                            "start": 4978,
                            "end": 4994
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4975,
                          "end": 4994
                        }
                      }
                    ],
                    "loc": {
                      "start": 4965,
                      "end": 5000
                    }
                  },
                  "loc": {
                    "start": 4945,
                    "end": 5000
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
                        "start": 5012,
                        "end": 5016
                      }
                    },
                    "loc": {
                      "start": 5012,
                      "end": 5016
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
                            "start": 5030,
                            "end": 5038
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5027,
                          "end": 5038
                        }
                      }
                    ],
                    "loc": {
                      "start": 5017,
                      "end": 5044
                    }
                  },
                  "loc": {
                    "start": 5005,
                    "end": 5044
                  }
                }
              ],
              "loc": {
                "start": 4939,
                "end": 5046
              }
            },
            "loc": {
              "start": 4933,
              "end": 5046
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 5047,
                "end": 5058
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5047,
              "end": 5058
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 5059,
                "end": 5073
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5059,
              "end": 5073
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 5074,
                "end": 5079
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5074,
              "end": 5079
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 5080,
                "end": 5089
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5080,
              "end": 5089
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 5090,
                "end": 5094
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
                      "start": 5104,
                      "end": 5112
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5101,
                    "end": 5112
                  }
                }
              ],
              "loc": {
                "start": 5095,
                "end": 5114
              }
            },
            "loc": {
              "start": 5090,
              "end": 5114
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 5115,
                "end": 5129
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5115,
              "end": 5129
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 5130,
                "end": 5135
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5130,
              "end": 5135
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 5136,
                "end": 5139
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
                      "start": 5146,
                      "end": 5156
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5146,
                    "end": 5156
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canDelete",
                    "loc": {
                      "start": 5161,
                      "end": 5170
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5161,
                    "end": 5170
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 5175,
                      "end": 5186
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5175,
                    "end": 5186
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 5191,
                      "end": 5200
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5191,
                    "end": 5200
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 5205,
                      "end": 5212
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5205,
                    "end": 5212
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 5217,
                      "end": 5225
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5217,
                    "end": 5225
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 5230,
                      "end": 5242
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5230,
                    "end": 5242
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 5247,
                      "end": 5255
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5247,
                    "end": 5255
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 5260,
                      "end": 5268
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5260,
                    "end": 5268
                  }
                }
              ],
              "loc": {
                "start": 5140,
                "end": 5270
              }
            },
            "loc": {
              "start": 5136,
              "end": 5270
            }
          }
        ],
        "loc": {
          "start": 4326,
          "end": 5272
        }
      },
      "loc": {
        "start": 4293,
        "end": 5272
      }
    },
    "Routine_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Routine_nav",
        "loc": {
          "start": 5282,
          "end": 5293
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Routine",
          "loc": {
            "start": 5297,
            "end": 5304
          }
        },
        "loc": {
          "start": 5297,
          "end": 5304
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
                "start": 5307,
                "end": 5309
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5307,
              "end": 5309
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isInternal",
              "loc": {
                "start": 5310,
                "end": 5320
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5310,
              "end": 5320
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5321,
                "end": 5330
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5321,
              "end": 5330
            }
          }
        ],
        "loc": {
          "start": 5305,
          "end": 5332
        }
      },
      "loc": {
        "start": 5273,
        "end": 5332
      }
    },
    "SmartContract_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "SmartContract_list",
        "loc": {
          "start": 5342,
          "end": 5360
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "SmartContract",
          "loc": {
            "start": 5364,
            "end": 5377
          }
        },
        "loc": {
          "start": 5364,
          "end": 5377
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
                "start": 5380,
                "end": 5388
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
                      "start": 5395,
                      "end": 5407
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
                            "start": 5418,
                            "end": 5420
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5418,
                          "end": 5420
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 5429,
                            "end": 5437
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5429,
                          "end": 5437
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 5446,
                            "end": 5457
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5446,
                          "end": 5457
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 5466,
                            "end": 5478
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5466,
                          "end": 5478
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 5487,
                            "end": 5491
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5487,
                          "end": 5491
                        }
                      }
                    ],
                    "loc": {
                      "start": 5408,
                      "end": 5497
                    }
                  },
                  "loc": {
                    "start": 5395,
                    "end": 5497
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5502,
                      "end": 5504
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5502,
                    "end": 5504
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 5509,
                      "end": 5519
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5509,
                    "end": 5519
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 5524,
                      "end": 5534
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5524,
                    "end": 5534
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 5539,
                      "end": 5549
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5539,
                    "end": 5549
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isDeleted",
                    "loc": {
                      "start": 5554,
                      "end": 5563
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5554,
                    "end": 5563
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 5568,
                      "end": 5576
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5568,
                    "end": 5576
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 5581,
                      "end": 5590
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5581,
                    "end": 5590
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 5595,
                      "end": 5602
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5595,
                    "end": 5602
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "contractType",
                    "loc": {
                      "start": 5607,
                      "end": 5619
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5607,
                    "end": 5619
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "content",
                    "loc": {
                      "start": 5624,
                      "end": 5631
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5624,
                    "end": 5631
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 5636,
                      "end": 5648
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5636,
                    "end": 5648
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 5653,
                      "end": 5665
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5653,
                    "end": 5665
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 5670,
                      "end": 5683
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5670,
                    "end": 5683
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 5688,
                      "end": 5710
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5688,
                    "end": 5710
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 5715,
                      "end": 5725
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5715,
                    "end": 5725
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 5730,
                      "end": 5742
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5730,
                    "end": 5742
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 5747,
                      "end": 5750
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
                            "start": 5761,
                            "end": 5771
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5761,
                          "end": 5771
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 5780,
                            "end": 5787
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5780,
                          "end": 5787
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 5796,
                            "end": 5805
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5796,
                          "end": 5805
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 5814,
                            "end": 5823
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5814,
                          "end": 5823
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 5832,
                            "end": 5841
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5832,
                          "end": 5841
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 5850,
                            "end": 5856
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5850,
                          "end": 5856
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 5865,
                            "end": 5872
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5865,
                          "end": 5872
                        }
                      }
                    ],
                    "loc": {
                      "start": 5751,
                      "end": 5878
                    }
                  },
                  "loc": {
                    "start": 5747,
                    "end": 5878
                  }
                }
              ],
              "loc": {
                "start": 5389,
                "end": 5880
              }
            },
            "loc": {
              "start": 5380,
              "end": 5880
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5881,
                "end": 5883
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5881,
              "end": 5883
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 5884,
                "end": 5894
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5884,
              "end": 5894
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 5895,
                "end": 5905
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5895,
              "end": 5905
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 5906,
                "end": 5915
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5906,
              "end": 5915
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 5916,
                "end": 5927
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5916,
              "end": 5927
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 5928,
                "end": 5934
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
                      "start": 5944,
                      "end": 5954
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5941,
                    "end": 5954
                  }
                }
              ],
              "loc": {
                "start": 5935,
                "end": 5956
              }
            },
            "loc": {
              "start": 5928,
              "end": 5956
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 5957,
                "end": 5962
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
                        "start": 5976,
                        "end": 5988
                      }
                    },
                    "loc": {
                      "start": 5976,
                      "end": 5988
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
                            "start": 6002,
                            "end": 6018
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 5999,
                          "end": 6018
                        }
                      }
                    ],
                    "loc": {
                      "start": 5989,
                      "end": 6024
                    }
                  },
                  "loc": {
                    "start": 5969,
                    "end": 6024
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
                        "start": 6036,
                        "end": 6040
                      }
                    },
                    "loc": {
                      "start": 6036,
                      "end": 6040
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
                            "start": 6054,
                            "end": 6062
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 6051,
                          "end": 6062
                        }
                      }
                    ],
                    "loc": {
                      "start": 6041,
                      "end": 6068
                    }
                  },
                  "loc": {
                    "start": 6029,
                    "end": 6068
                  }
                }
              ],
              "loc": {
                "start": 5963,
                "end": 6070
              }
            },
            "loc": {
              "start": 5957,
              "end": 6070
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 6071,
                "end": 6082
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6071,
              "end": 6082
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 6083,
                "end": 6097
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6083,
              "end": 6097
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 6098,
                "end": 6103
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6098,
              "end": 6103
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 6104,
                "end": 6113
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6104,
              "end": 6113
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 6114,
                "end": 6118
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
                      "start": 6128,
                      "end": 6136
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6125,
                    "end": 6136
                  }
                }
              ],
              "loc": {
                "start": 6119,
                "end": 6138
              }
            },
            "loc": {
              "start": 6114,
              "end": 6138
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 6139,
                "end": 6153
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6139,
              "end": 6153
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 6154,
                "end": 6159
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6154,
              "end": 6159
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 6160,
                "end": 6163
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
                      "start": 6170,
                      "end": 6179
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6170,
                    "end": 6179
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 6184,
                      "end": 6195
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6184,
                    "end": 6195
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 6200,
                      "end": 6211
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6200,
                    "end": 6211
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 6216,
                      "end": 6225
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6216,
                    "end": 6225
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 6230,
                      "end": 6237
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6230,
                    "end": 6237
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 6242,
                      "end": 6250
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6242,
                    "end": 6250
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 6255,
                      "end": 6267
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6255,
                    "end": 6267
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 6272,
                      "end": 6280
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6272,
                    "end": 6280
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 6285,
                      "end": 6293
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6285,
                    "end": 6293
                  }
                }
              ],
              "loc": {
                "start": 6164,
                "end": 6295
              }
            },
            "loc": {
              "start": 6160,
              "end": 6295
            }
          }
        ],
        "loc": {
          "start": 5378,
          "end": 6297
        }
      },
      "loc": {
        "start": 5333,
        "end": 6297
      }
    },
    "SmartContract_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "SmartContract_nav",
        "loc": {
          "start": 6307,
          "end": 6324
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "SmartContract",
          "loc": {
            "start": 6328,
            "end": 6341
          }
        },
        "loc": {
          "start": 6328,
          "end": 6341
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
                "start": 6344,
                "end": 6346
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6344,
              "end": 6346
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6347,
                "end": 6356
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6347,
              "end": 6356
            }
          }
        ],
        "loc": {
          "start": 6342,
          "end": 6358
        }
      },
      "loc": {
        "start": 6298,
        "end": 6358
      }
    },
    "Standard_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_list",
        "loc": {
          "start": 6368,
          "end": 6381
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 6385,
            "end": 6393
          }
        },
        "loc": {
          "start": 6385,
          "end": 6393
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
                "start": 6396,
                "end": 6404
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
                      "start": 6411,
                      "end": 6423
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
                            "start": 6434,
                            "end": 6436
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6434,
                          "end": 6436
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "language",
                          "loc": {
                            "start": 6445,
                            "end": 6453
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6445,
                          "end": 6453
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 6462,
                            "end": 6473
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6462,
                          "end": 6473
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "jsonVariable",
                          "loc": {
                            "start": 6482,
                            "end": 6494
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6482,
                          "end": 6494
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 6503,
                            "end": 6507
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6503,
                          "end": 6507
                        }
                      }
                    ],
                    "loc": {
                      "start": 6424,
                      "end": 6513
                    }
                  },
                  "loc": {
                    "start": 6411,
                    "end": 6513
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 6518,
                      "end": 6520
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6518,
                    "end": 6520
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 6525,
                      "end": 6535
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6525,
                    "end": 6535
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 6540,
                      "end": 6550
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6540,
                    "end": 6550
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isComplete",
                    "loc": {
                      "start": 6555,
                      "end": 6565
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6555,
                    "end": 6565
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isFile",
                    "loc": {
                      "start": 6570,
                      "end": 6576
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6570,
                    "end": 6576
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isLatest",
                    "loc": {
                      "start": 6581,
                      "end": 6589
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6581,
                    "end": 6589
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isPrivate",
                    "loc": {
                      "start": 6594,
                      "end": 6603
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6594,
                    "end": 6603
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "default",
                    "loc": {
                      "start": 6608,
                      "end": 6615
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6608,
                    "end": 6615
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardType",
                    "loc": {
                      "start": 6620,
                      "end": 6632
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6620,
                    "end": 6632
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "props",
                    "loc": {
                      "start": 6637,
                      "end": 6642
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6637,
                    "end": 6642
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "yup",
                    "loc": {
                      "start": 6647,
                      "end": 6650
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6647,
                    "end": 6650
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionIndex",
                    "loc": {
                      "start": 6655,
                      "end": 6667
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6655,
                    "end": 6667
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "versionLabel",
                    "loc": {
                      "start": 6672,
                      "end": 6684
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6672,
                    "end": 6684
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "commentsCount",
                    "loc": {
                      "start": 6689,
                      "end": 6702
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6689,
                    "end": 6702
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "directoryListingsCount",
                    "loc": {
                      "start": 6707,
                      "end": 6729
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6707,
                    "end": 6729
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "forksCount",
                    "loc": {
                      "start": 6734,
                      "end": 6744
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6734,
                    "end": 6744
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reportsCount",
                    "loc": {
                      "start": 6749,
                      "end": 6761
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 6749,
                    "end": 6761
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 6766,
                      "end": 6769
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
                            "start": 6780,
                            "end": 6790
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6780,
                          "end": 6790
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canCopy",
                          "loc": {
                            "start": 6799,
                            "end": 6806
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6799,
                          "end": 6806
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canDelete",
                          "loc": {
                            "start": 6815,
                            "end": 6824
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6815,
                          "end": 6824
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canReport",
                          "loc": {
                            "start": 6833,
                            "end": 6842
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6833,
                          "end": 6842
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 6851,
                            "end": 6860
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6851,
                          "end": 6860
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUse",
                          "loc": {
                            "start": 6869,
                            "end": 6875
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6869,
                          "end": 6875
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 6884,
                            "end": 6891
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 6884,
                          "end": 6891
                        }
                      }
                    ],
                    "loc": {
                      "start": 6770,
                      "end": 6897
                    }
                  },
                  "loc": {
                    "start": 6766,
                    "end": 6897
                  }
                }
              ],
              "loc": {
                "start": 6405,
                "end": 6899
              }
            },
            "loc": {
              "start": 6396,
              "end": 6899
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 6900,
                "end": 6902
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6900,
              "end": 6902
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 6903,
                "end": 6913
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6903,
              "end": 6913
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 6914,
                "end": 6924
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6914,
              "end": 6924
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 6925,
                "end": 6934
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6925,
              "end": 6934
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "issuesCount",
              "loc": {
                "start": 6935,
                "end": 6946
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 6935,
              "end": 6946
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "labels",
              "loc": {
                "start": 6947,
                "end": 6953
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
                      "start": 6963,
                      "end": 6973
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 6960,
                    "end": 6973
                  }
                }
              ],
              "loc": {
                "start": 6954,
                "end": 6975
              }
            },
            "loc": {
              "start": 6947,
              "end": 6975
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "owner",
              "loc": {
                "start": 6976,
                "end": 6981
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
                        "start": 6995,
                        "end": 7007
                      }
                    },
                    "loc": {
                      "start": 6995,
                      "end": 7007
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
                            "start": 7021,
                            "end": 7037
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7018,
                          "end": 7037
                        }
                      }
                    ],
                    "loc": {
                      "start": 7008,
                      "end": 7043
                    }
                  },
                  "loc": {
                    "start": 6988,
                    "end": 7043
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
                        "start": 7055,
                        "end": 7059
                      }
                    },
                    "loc": {
                      "start": 7055,
                      "end": 7059
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
                            "start": 7073,
                            "end": 7081
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 7070,
                          "end": 7081
                        }
                      }
                    ],
                    "loc": {
                      "start": 7060,
                      "end": 7087
                    }
                  },
                  "loc": {
                    "start": 7048,
                    "end": 7087
                  }
                }
              ],
              "loc": {
                "start": 6982,
                "end": 7089
              }
            },
            "loc": {
              "start": 6976,
              "end": 7089
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "permissions",
              "loc": {
                "start": 7090,
                "end": 7101
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7090,
              "end": 7101
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsCount",
              "loc": {
                "start": 7102,
                "end": 7116
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7102,
              "end": 7116
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "score",
              "loc": {
                "start": 7117,
                "end": 7122
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7117,
              "end": 7122
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7123,
                "end": 7132
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7123,
              "end": 7132
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tags",
              "loc": {
                "start": 7133,
                "end": 7137
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
                      "start": 7147,
                      "end": 7155
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 7144,
                    "end": 7155
                  }
                }
              ],
              "loc": {
                "start": 7138,
                "end": 7157
              }
            },
            "loc": {
              "start": 7133,
              "end": 7157
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "transfersCount",
              "loc": {
                "start": 7158,
                "end": 7172
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7158,
              "end": 7172
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "views",
              "loc": {
                "start": 7173,
                "end": 7178
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7173,
              "end": 7178
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7179,
                "end": 7182
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
                      "start": 7189,
                      "end": 7198
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7189,
                    "end": 7198
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canBookmark",
                    "loc": {
                      "start": 7203,
                      "end": 7214
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7203,
                    "end": 7214
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canTransfer",
                    "loc": {
                      "start": 7219,
                      "end": 7230
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7219,
                    "end": 7230
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7235,
                      "end": 7244
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7235,
                    "end": 7244
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canRead",
                    "loc": {
                      "start": 7249,
                      "end": 7256
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7249,
                    "end": 7256
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReact",
                    "loc": {
                      "start": 7261,
                      "end": 7269
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7261,
                    "end": 7269
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7274,
                      "end": 7286
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7274,
                    "end": 7286
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7291,
                      "end": 7299
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7291,
                    "end": 7299
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reaction",
                    "loc": {
                      "start": 7304,
                      "end": 7312
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7304,
                    "end": 7312
                  }
                }
              ],
              "loc": {
                "start": 7183,
                "end": 7314
              }
            },
            "loc": {
              "start": 7179,
              "end": 7314
            }
          }
        ],
        "loc": {
          "start": 6394,
          "end": 7316
        }
      },
      "loc": {
        "start": 6359,
        "end": 7316
      }
    },
    "Standard_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Standard_nav",
        "loc": {
          "start": 7326,
          "end": 7338
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Standard",
          "loc": {
            "start": 7342,
            "end": 7350
          }
        },
        "loc": {
          "start": 7342,
          "end": 7350
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
                "start": 7353,
                "end": 7355
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7353,
              "end": 7355
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isPrivate",
              "loc": {
                "start": 7356,
                "end": 7365
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7356,
              "end": 7365
            }
          }
        ],
        "loc": {
          "start": 7351,
          "end": 7367
        }
      },
      "loc": {
        "start": 7317,
        "end": 7367
      }
    },
    "Tag_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Tag_list",
        "loc": {
          "start": 7377,
          "end": 7385
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Tag",
          "loc": {
            "start": 7389,
            "end": 7392
          }
        },
        "loc": {
          "start": 7389,
          "end": 7392
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
                "start": 7395,
                "end": 7397
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7395,
              "end": 7397
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7398,
                "end": 7408
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7398,
              "end": 7408
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "tag",
              "loc": {
                "start": 7409,
                "end": 7412
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7409,
              "end": 7412
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7413,
                "end": 7422
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7413,
              "end": 7422
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "translations",
              "loc": {
                "start": 7423,
                "end": 7435
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
                      "start": 7442,
                      "end": 7444
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7442,
                    "end": 7444
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
                    "loc": {
                      "start": 7449,
                      "end": 7457
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7449,
                    "end": 7457
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 7462,
                      "end": 7473
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7462,
                    "end": 7473
                  }
                }
              ],
              "loc": {
                "start": 7436,
                "end": 7475
              }
            },
            "loc": {
              "start": 7423,
              "end": 7475
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7476,
                "end": 7479
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
                      "start": 7486,
                      "end": 7491
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7486,
                    "end": 7491
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7496,
                      "end": 7508
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7496,
                    "end": 7508
                  }
                }
              ],
              "loc": {
                "start": 7480,
                "end": 7510
              }
            },
            "loc": {
              "start": 7476,
              "end": 7510
            }
          }
        ],
        "loc": {
          "start": 7393,
          "end": 7512
        }
      },
      "loc": {
        "start": 7368,
        "end": 7512
      }
    },
    "User_list": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_list",
        "loc": {
          "start": 7522,
          "end": 7531
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7535,
            "end": 7539
          }
        },
        "loc": {
          "start": 7535,
          "end": 7539
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
                "start": 7542,
                "end": 7554
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
                      "start": 7561,
                      "end": 7563
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7561,
                    "end": 7563
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "language",
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
                    "value": "bio",
                    "loc": {
                      "start": 7581,
                      "end": 7584
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7581,
                    "end": 7584
                  }
                }
              ],
              "loc": {
                "start": 7555,
                "end": 7586
              }
            },
            "loc": {
              "start": 7542,
              "end": 7586
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 7587,
                "end": 7589
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7587,
              "end": 7589
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 7590,
                "end": 7600
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7590,
              "end": 7600
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7601,
                "end": 7612
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7601,
              "end": 7612
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7613,
                "end": 7619
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7613,
              "end": 7619
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7620,
                "end": 7625
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7620,
              "end": 7625
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7626,
                "end": 7630
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7626,
              "end": 7630
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7631,
                "end": 7643
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7631,
              "end": 7643
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarks",
              "loc": {
                "start": 7644,
                "end": 7653
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7644,
              "end": 7653
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "reportsReceivedCount",
              "loc": {
                "start": 7654,
                "end": 7674
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7654,
              "end": 7674
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "you",
              "loc": {
                "start": 7675,
                "end": 7678
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
                      "start": 7685,
                      "end": 7694
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7685,
                    "end": 7694
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canReport",
                    "loc": {
                      "start": 7699,
                      "end": 7708
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7699,
                    "end": 7708
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "canUpdate",
                    "loc": {
                      "start": 7713,
                      "end": 7722
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7713,
                    "end": 7722
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isBookmarked",
                    "loc": {
                      "start": 7727,
                      "end": 7739
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7727,
                    "end": 7739
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "isViewed",
                    "loc": {
                      "start": 7744,
                      "end": 7752
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 7744,
                    "end": 7752
                  }
                }
              ],
              "loc": {
                "start": 7679,
                "end": 7754
              }
            },
            "loc": {
              "start": 7675,
              "end": 7754
            }
          }
        ],
        "loc": {
          "start": 7540,
          "end": 7756
        }
      },
      "loc": {
        "start": 7513,
        "end": 7756
      }
    },
    "User_nav": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "User_nav",
        "loc": {
          "start": 7766,
          "end": 7774
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "User",
          "loc": {
            "start": 7778,
            "end": 7782
          }
        },
        "loc": {
          "start": 7778,
          "end": 7782
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
                "start": 7785,
                "end": 7787
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7785,
              "end": 7787
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bannerImage",
              "loc": {
                "start": 7788,
                "end": 7799
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7788,
              "end": 7799
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 7800,
                "end": 7806
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7800,
              "end": 7806
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isBot",
              "loc": {
                "start": 7807,
                "end": 7812
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7807,
              "end": 7812
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 7813,
                "end": 7817
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7813,
              "end": 7817
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "profileImage",
              "loc": {
                "start": 7818,
                "end": 7830
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 7818,
              "end": 7830
            }
          }
        ],
        "loc": {
          "start": 7783,
          "end": 7832
        }
      },
      "loc": {
        "start": 7757,
        "end": 7832
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "query",
    "name": {
      "kind": "Name",
      "value": "populars",
      "loc": {
        "start": 7840,
        "end": 7848
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
              "start": 7850,
              "end": 7855
            }
          },
          "loc": {
            "start": 7849,
            "end": 7855
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "PopularSearchInput",
              "loc": {
                "start": 7857,
                "end": 7875
              }
            },
            "loc": {
              "start": 7857,
              "end": 7875
            }
          },
          "loc": {
            "start": 7857,
            "end": 7876
          }
        },
        "directives": [],
        "loc": {
          "start": 7849,
          "end": 7876
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
            "value": "populars",
            "loc": {
              "start": 7882,
              "end": 7890
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 7891,
                  "end": 7896
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 7899,
                    "end": 7904
                  }
                },
                "loc": {
                  "start": 7898,
                  "end": 7904
                }
              },
              "loc": {
                "start": 7891,
                "end": 7904
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
                    "start": 7912,
                    "end": 7917
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
                          "start": 7928,
                          "end": 7934
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 7928,
                        "end": 7934
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "node",
                        "loc": {
                          "start": 7943,
                          "end": 7947
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
                                  "start": 7969,
                                  "end": 7972
                                }
                              },
                              "loc": {
                                "start": 7969,
                                "end": 7972
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
                                    "value": "Api_list",
                                    "loc": {
                                      "start": 7994,
                                      "end": 8002
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 7991,
                                    "end": 8002
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7973,
                                "end": 8016
                              }
                            },
                            "loc": {
                              "start": 7962,
                              "end": 8016
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
                                  "start": 8036,
                                  "end": 8040
                                }
                              },
                              "loc": {
                                "start": 8036,
                                "end": 8040
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
                                    "value": "Note_list",
                                    "loc": {
                                      "start": 8062,
                                      "end": 8071
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8059,
                                    "end": 8071
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8041,
                                "end": 8085
                              }
                            },
                            "loc": {
                              "start": 8029,
                              "end": 8085
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
                                  "start": 8105,
                                  "end": 8117
                                }
                              },
                              "loc": {
                                "start": 8105,
                                "end": 8117
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
                                    "value": "Organization_list",
                                    "loc": {
                                      "start": 8139,
                                      "end": 8156
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8136,
                                    "end": 8156
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8118,
                                "end": 8170
                              }
                            },
                            "loc": {
                              "start": 8098,
                              "end": 8170
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
                                  "start": 8190,
                                  "end": 8197
                                }
                              },
                              "loc": {
                                "start": 8190,
                                "end": 8197
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
                                    "value": "Project_list",
                                    "loc": {
                                      "start": 8219,
                                      "end": 8231
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8216,
                                    "end": 8231
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8198,
                                "end": 8245
                              }
                            },
                            "loc": {
                              "start": 8183,
                              "end": 8245
                            }
                          },
                          {
                            "kind": "InlineFragment",
                            "typeCondition": {
                              "kind": "NamedType",
                              "name": {
                                "kind": "Name",
                                "value": "Question",
                                "loc": {
                                  "start": 8265,
                                  "end": 8273
                                }
                              },
                              "loc": {
                                "start": 8265,
                                "end": 8273
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
                                    "value": "Question_list",
                                    "loc": {
                                      "start": 8295,
                                      "end": 8308
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8292,
                                    "end": 8308
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8274,
                                "end": 8322
                              }
                            },
                            "loc": {
                              "start": 8258,
                              "end": 8322
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
                                  "start": 8342,
                                  "end": 8349
                                }
                              },
                              "loc": {
                                "start": 8342,
                                "end": 8349
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
                                    "value": "Routine_list",
                                    "loc": {
                                      "start": 8371,
                                      "end": 8383
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8368,
                                    "end": 8383
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8350,
                                "end": 8397
                              }
                            },
                            "loc": {
                              "start": 8335,
                              "end": 8397
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
                                  "start": 8417,
                                  "end": 8430
                                }
                              },
                              "loc": {
                                "start": 8417,
                                "end": 8430
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
                                    "value": "SmartContract_list",
                                    "loc": {
                                      "start": 8452,
                                      "end": 8470
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8449,
                                    "end": 8470
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8431,
                                "end": 8484
                              }
                            },
                            "loc": {
                              "start": 8410,
                              "end": 8484
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
                                  "start": 8504,
                                  "end": 8512
                                }
                              },
                              "loc": {
                                "start": 8504,
                                "end": 8512
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
                                    "value": "Standard_list",
                                    "loc": {
                                      "start": 8534,
                                      "end": 8547
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8531,
                                    "end": 8547
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8513,
                                "end": 8561
                              }
                            },
                            "loc": {
                              "start": 8497,
                              "end": 8561
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
                                  "start": 8581,
                                  "end": 8585
                                }
                              },
                              "loc": {
                                "start": 8581,
                                "end": 8585
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
                                    "value": "User_list",
                                    "loc": {
                                      "start": 8607,
                                      "end": 8616
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 8604,
                                    "end": 8616
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8586,
                                "end": 8630
                              }
                            },
                            "loc": {
                              "start": 8574,
                              "end": 8630
                            }
                          }
                        ],
                        "loc": {
                          "start": 7948,
                          "end": 8640
                        }
                      },
                      "loc": {
                        "start": 7943,
                        "end": 8640
                      }
                    }
                  ],
                  "loc": {
                    "start": 7918,
                    "end": 8646
                  }
                },
                "loc": {
                  "start": 7912,
                  "end": 8646
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "pageInfo",
                  "loc": {
                    "start": 8651,
                    "end": 8659
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
                        "value": "hasNextPage",
                        "loc": {
                          "start": 8670,
                          "end": 8681
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8670,
                        "end": 8681
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorApi",
                        "loc": {
                          "start": 8690,
                          "end": 8702
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8690,
                        "end": 8702
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorNote",
                        "loc": {
                          "start": 8711,
                          "end": 8724
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8711,
                        "end": 8724
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorOrganization",
                        "loc": {
                          "start": 8733,
                          "end": 8754
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8733,
                        "end": 8754
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorProject",
                        "loc": {
                          "start": 8763,
                          "end": 8779
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8763,
                        "end": 8779
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorQuestion",
                        "loc": {
                          "start": 8788,
                          "end": 8805
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8788,
                        "end": 8805
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorRoutine",
                        "loc": {
                          "start": 8814,
                          "end": 8830
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8814,
                        "end": 8830
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorSmartContract",
                        "loc": {
                          "start": 8839,
                          "end": 8861
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8839,
                        "end": 8861
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorStandard",
                        "loc": {
                          "start": 8870,
                          "end": 8887
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8870,
                        "end": 8887
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "endCursorUser",
                        "loc": {
                          "start": 8896,
                          "end": 8909
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 8896,
                        "end": 8909
                      }
                    }
                  ],
                  "loc": {
                    "start": 8660,
                    "end": 8915
                  }
                },
                "loc": {
                  "start": 8651,
                  "end": 8915
                }
              }
            ],
            "loc": {
              "start": 7906,
              "end": 8919
            }
          },
          "loc": {
            "start": 7882,
            "end": 8919
          }
        }
      ],
      "loc": {
        "start": 7878,
        "end": 8921
      }
    },
    "loc": {
      "start": 7834,
      "end": 8921
    }
  },
  "variableValues": {},
  "path": {
    "key": "popular_findMany"
  }
} as const;
